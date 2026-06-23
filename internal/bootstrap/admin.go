// Package bootstrap 负责首次启动时的一次性初始化逻辑,目前包含默认管理员的创建。
//
// 安全约定(见 docs/SECURITY.md 与 docs/DECISIONS/0008-admin-password-hashing.md):
//   - 默认管理员密码强随机生成,只在创建时一次性返回供打印,绝不持久化、绝不写日志文件。
//   - 数据库中只保存 bcrypt 哈希,不保存明文。
//   - 新建管理员的 must_change_password 为 true,要求其首次登录后修改密码。
//   - 仅当 admins 表为空时才创建;已存在管理员则不创建、不返回密码。
package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/fengheasia/frp-warden/internal/security"
)

// AdminInitResult 描述一次管理员初始化的结果。
type AdminInitResult struct {
	// Created 表示本次是否新建了默认管理员。
	Created bool
	// Username 为新建管理员的用户名(仅在 Created 为 true 时有意义)。
	Username string
	// Password 为新建管理员的明文密码,仅存在于内存中,供调用方一次性打印。
	// 绝不持久化、绝不写入日志文件或配置文件。仅在 Created 为 true 时有效。
	Password string
}

// EnsureInitialAdmin 在 admins 表为空时创建默认管理员。
//
// 若已存在任意管理员,则不创建、不生成密码,返回 Created=false。
func EnsureInitialAdmin(ctx context.Context, sdb *sql.DB, username string) (AdminInitResult, error) {
	var count int
	if err := sdb.QueryRowContext(ctx, `SELECT COUNT(*) FROM admins`).Scan(&count); err != nil {
		return AdminInitResult{}, fmt.Errorf("bootstrap: 查询管理员数量失败: %w", err)
	}
	if count > 0 {
		return AdminInitResult{Created: false}, nil
	}

	password, err := security.GenerateRandomPassword(0)
	if err != nil {
		return AdminInitResult{}, err
	}
	hash, err := security.HashPassword(password)
	if err != nil {
		return AdminInitResult{}, err
	}

	tx, err := sdb.BeginTx(ctx, nil)
	if err != nil {
		return AdminInitResult{}, fmt.Errorf("bootstrap: 开启事务失败: %w", err)
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO admins (username, password_hash, must_change_password, status)
		 VALUES (?, ?, 1, 'enabled')`, username, hash); err != nil {
		_ = tx.Rollback()
		return AdminInitResult{}, fmt.Errorf("bootstrap: 创建默认管理员失败: %w", err)
	}

	// 在 settings 中记录初始化时间,示范 settings 表"保存初始化状态"的用途。
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO settings (key, value) VALUES (?, ?)`,
		"initial_admin_created_at", time.Now().UTC().Format(time.RFC3339)); err != nil {
		_ = tx.Rollback()
		return AdminInitResult{}, fmt.Errorf("bootstrap: 记录初始化状态失败: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return AdminInitResult{}, fmt.Errorf("bootstrap: 提交事务失败: %w", err)
	}

	return AdminInitResult{Created: true, Username: username, Password: password}, nil
}
