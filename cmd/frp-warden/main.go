// Command frp-warden 是 frp-warden 控制面的入口程序。
//
// 截至 Phase 5:在配置(Phase 1)、数据库/迁移/管理员初始化(Phase 2)、
// tenant/resource/grant/proxy 数据模型(Phase 3)、frps plugin 鉴权(Phase 4)之上,
// Phase 5 实现了管理后台 REST API(见 internal/admin)与基于 session cookie 的
// 管理员认证。默认启动 HTTP 服务;使用 -version 打印版本后退出。
//
// 部署拓扑:frps 与 frp-warden 都部署在公网服务器/VPS;plugin 接口默认仅监听 127.0.0.1。
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/fengheasia/frp-warden/internal/bootstrap"
	"github.com/fengheasia/frp-warden/internal/config"
	"github.com/fengheasia/frp-warden/internal/db"
	"github.com/fengheasia/frp-warden/internal/server"
	"github.com/fengheasia/frp-warden/internal/store"
	"github.com/fengheasia/frp-warden/internal/version"
)

func main() {
	var (
		showVersion bool
		configPath  string
	)
	flag.BoolVar(&showVersion, "version", false, "打印版本信息后退出")
	flag.StringVar(&configPath, "config", config.DefaultConfigPath, "配置文件路径")
	flag.StringVar(&configPath, "c", config.DefaultConfigPath, "配置文件路径（-config 的简写）")
	flag.Parse()

	if showVersion {
		fmt.Println(version.String())
		return
	}

	// 1) 加载配置。
	res, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	cfg := res.Config

	// 2) 确保数据目录存在。
	if err := config.EnsureDataDir(cfg); err != nil {
		log.Fatalf("初始化数据目录失败: %v", err)
	}

	// 3) 打开数据库。
	sdb, dialect, err := db.Open(cfg.Database)
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
	}
	defer sdb.Close()

	// 4) 迁移。
	ctx := context.Background()
	migrated, err := db.Migrate(ctx, sdb, dialect)
	if err != nil {
		log.Fatalf("执行数据库迁移失败: %v", err)
	}

	// 5) 初始化默认管理员。
	adminInit, err := bootstrap.EnsureInitialAdmin(ctx, sdb, cfg.Security.InitialAdminUsername)
	if err != nil {
		log.Fatalf("初始化默认管理员失败: %v", err)
	}
	if adminInit.Created {
		printInitialAdmin(adminInit.Username, adminInit.Password)
	}

	// 6) 启动摘要。
	printStartupSummary(res, cfg, migrated, adminInit.Created)

	// 7) 启动 HTTP 服务(admin + plugin)。
	st := store.New(sdb)
	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println("正在启动 HTTP 服务...")
	if err := server.New(cfg, st).Run(runCtx); err != nil {
		log.Fatalf("服务运行出错: %v", err)
	}
}

func printInitialAdmin(username, password string) {
	const line = "============================================================"
	fmt.Println(line)
	fmt.Println("frp-warden 已创建默认管理员账号")
	fmt.Println()
	fmt.Printf("用户名: %s\n", username)
	fmt.Printf("密码: %s\n", password)
	fmt.Println()
	fmt.Println("请立即登录管理后台修改默认密码。")
	fmt.Println("该密码只会显示一次，请妥善保存。")
	fmt.Println(line)
}

func printStartupSummary(res *config.LoadResult, cfg config.Config, migrated int, adminCreated bool) {
	fmt.Println(version.String())
	if res.Generated {
		fmt.Printf("未找到配置文件，已自动生成默认配置: %s\n", res.Path)
	}
	if res.SecretGenerated {
		fmt.Println("session_secret 为空，已自动生成强随机值并写回配置文件")
	}
	fmt.Printf("配置文件: %s\n", res.Path)
	fmt.Printf("database.driver: %s\n", cfg.Database.Driver)
	fmt.Printf("database.dsn: %s\n", cfg.Database.DSN)
	fmt.Printf("本次应用迁移数: %d\n", migrated)
	if adminCreated {
		fmt.Println("默认管理员: 本次已创建（密码见上方，仅显示一次）")
	} else {
		fmt.Println("默认管理员: 已存在，未重复创建")
	}
	fmt.Printf("admin_addr: %s\n", cfg.Server.AdminAddr)
	fmt.Printf("plugin_addr: %s\n", cfg.Server.PluginAddr)
}
