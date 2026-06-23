package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/fengheasia/frp-warden/internal/model"
)

const adminColumns = "id, username, password_hash, must_change_password, status"

func scanAdmin(s rowScanner) (model.Admin, error) {
	var a model.Admin
	err := s.Scan(&a.ID, &a.Username, &a.PasswordHash, &a.MustChangePassword, &a.Status)
	return a, err
}

// GetAdminByUsername 按用户名查询管理员,不存在返回 ErrAdminNotFound。
func (s *Store) GetAdminByUsername(ctx context.Context, username string) (model.Admin, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+adminColumns+` FROM admins WHERE username = ?`, username)
	a, err := scanAdmin(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Admin{}, ErrAdminNotFound
	}
	return a, err
}

// GetAdminByID 按 id 查询管理员,不存在返回 ErrAdminNotFound。
func (s *Store) GetAdminByID(ctx context.Context, id int64) (model.Admin, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+adminColumns+` FROM admins WHERE id = ?`, id)
	a, err := scanAdmin(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Admin{}, ErrAdminNotFound
	}
	return a, err
}

// UpdateAdminPassword 更新管理员密码哈希,并将 must_change_password 置为 false。
// 管理员不存在返回 ErrAdminNotFound。
func (s *Store) UpdateAdminPassword(ctx context.Context, id int64, passwordHash string) error {
	if strings.TrimSpace(passwordHash) == "" {
		return ErrEmptyField
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE admins SET password_hash = ?, must_change_password = 0, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		passwordHash, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrAdminNotFound
	}
	return nil
}

// UpdateAdminLastLogin 更新管理员最近登录时间。
func (s *Store) UpdateAdminLastLogin(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE admins SET last_login_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}
