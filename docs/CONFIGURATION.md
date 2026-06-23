# 配置

frp-warden 使用单个 YAML 文件进行配置。`internal/config` 中的 Go struct 是配置
schema 的唯一来源,本文档必须与之保持同步。

## 默认配置

```yaml
server:
  admin_addr: "0.0.0.0:8080"
  plugin_addr: "127.0.0.1:9000"

database:
  driver: "sqlite"
  dsn: "./data/frp-warden.db"

security:
  session_secret: ""
  initial_admin_username: "admin"

frp:
  server_addr: "127.0.0.1"
  server_port: 7000
  subdomain_host: ""

log:
  level: "info"
```

> 说明:程序自动生成的 `config.yaml` 会在文件顶部附带说明注释,并填入实际生成的
> `session_secret`(因此磁盘上的文件中该字段不为空)。

## 字段说明

### server

| 键            | 默认值             | 含义                                                              |
|---------------|--------------------|-------------------------------------------------------------------|
| `admin_addr`  | `0.0.0.0:8080`     | 管理后台 + API 的监听地址。                                        |
| `plugin_addr` | `127.0.0.1:9000`   | frps plugin 接口监听地址。frps 与 frp-warden 同机/同内网部署,默认仅回环、只允许本机 frps 调用、不得暴露公网。         |

### database

| 键       | 默认值                   | 含义                                                       |
|----------|--------------------------|------------------------------------------------------------|
| `driver` | `sqlite`                 | 取值之一:`sqlite`、`mysql`、`mariadb`、`postgresql`。      |
| `dsn`    | `./data/frp-warden.db`   | 所选驱动的数据源 / 连接串。                                 |

默认使用 SQLite,并采用纯 Go(无 CGO)驱动 `modernc.org/sqlite`。各驱动当前成熟度:

| driver               | 状态                                                            |
|----------------------|-----------------------------------------------------------------|
| `sqlite`             | **完整可用**:连接、迁移、管理员初始化均已实现,测试覆盖。       |
| `mysql` / `mariadb`  | 仅连接骨架(已注册驱动),**迁移尚未实现**,执行迁移时 fail fast 报错。 |
| `postgresql`         | 仅连接骨架(已注册驱动),**迁移尚未实现**,执行迁移时 fail fast 报错。 |

> 即:目前请使用 `sqlite`。选择其它 driver 时程序能建立连接,但在迁移阶段会明确报错退出,
> 不会"看似可用实则不可用"。对应的 SQLite DSN 在未显式带查询参数时会自动追加
> `?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)`(开启外键约束与忙等待)。

### security

| 键                       | 默认值    | 含义                                                                                       |
|--------------------------|-----------|--------------------------------------------------------------------------------------------|
| `session_secret`         | `""`      | 会话/HMAC 签名密钥。**为空时首次启动自动生成强随机值并写回文件**(见下文首次启动流程)。     |
| `initial_admin_username` | `admin`   | 自动创建的初始管理员用户名。初始**密码在初始化时随机生成**,绝不写入配置(仅 hash 存入数据库,见 [SECURITY.md](SECURITY.md))。 |

### frp

用于生成 frpc 客户端配置。

| 键               | 默认值        | 含义                                          |
|------------------|---------------|-----------------------------------------------|
| `server_addr`    | `127.0.0.1`   | 客户端连接的 frps 地址。                       |
| `server_port`    | `7000`        | frps 控制端口。                                |
| `subdomain_host` | `""`          | subdomain proxy 使用的泛域名根。               |

### log

| 键      | 默认值   | 含义                                          |
|---------|----------|-----------------------------------------------|
| `level` | `info`   | 日志级别(`debug`/`info`/`warn`/`error`)。    |

## 命令行参数

| 参数             | 含义                                                       |
|------------------|------------------------------------------------------------|
| `-version`       | 打印版本信息后退出(不加载配置)。                          |
| `-c <path>`      | 指定配置文件路径(`-config` 的简写)。                      |
| `-config <path>` | 指定配置文件路径。默认 `./config.yaml`。                    |
| `-serve`         | 启动占位的 admin 与 plugin HTTP 服务(Phase 1 可选,默认不启动)。 |

示例:

```sh
frp-warden
frp-warden -c ./config.yaml
frp-warden -config ./config.yaml
frp-warden -version
```

## 首次启动行为(Phase 1,已实现)

1. 如果配置文件不存在,frp-warden 会把上述默认配置**写入磁盘**并继续运行。
2. 如果 `session_secret` 为空,会使用 `crypto/rand` 生成至少 32 字节熵的强随机密钥
   (base64 编码),并**写回**配置文件。
3. 配置文件存在时会被读取并解析;文件中未出现的键沿用默认值。
4. 若配置文件**格式错误**或**校验不通过**,程序会明确报错并退出,不会静默忽略。
5. 使用 SQLite 时,会根据 `database.dsn` 推断并**创建数据目录**(如 `./data`)。
6. **打开数据库连接**并执行 Ping 连通性检查(Phase 2,SQLite 已实现)。
7. **执行迁移**:确保 `schema_migrations` 存在,按版本顺序应用未执行的迁移,创建
   `admins`/`settings`/`sessions`/`audit_logs` 等基础表;迁移失败立即报错退出。
8. **初始化默认管理员**:当 `admins` 表为空时,以 `security.initial_admin_username`
   为用户名创建默认管理员,密码**强随机生成并一次性打印**到控制台,仅以 bcrypt hash 存库,
   `must_change_password=true`。已存在管理员时不创建、不打印。

## 配置校验规则(Phase 1)

启动时进行基础校验,非法即报错退出:

- `server.admin_addr`、`server.plugin_addr` 不能为空。
- `database.driver` 不能为空,且必须是 `sqlite`/`mysql`/`mariadb`/`postgresql` 之一。
- `database.dsn` 不能为空。
- `security.initial_admin_username` 不能为空。
- `frp.server_addr` 不能为空;`frp.server_port` 必须大于 0。
- `log.level` 必须是 `debug`/`info`/`warn`/`error` 之一。

## 安全提示

配置文件可能包含 `session_secret`,属于敏感信息,自动写入时文件权限为 `0600`。
请将配置文件视为敏感文件妥善保管。

启动摘要与日志**绝不打印** `session_secret`、`password_hash`、`token_hash` 或数据库密码;
默认管理员的明文密码只在首次创建时一次性打印,且不会写入配置文件或日志文件。
