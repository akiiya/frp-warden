package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/fengheasia/frp-warden/internal/model"
	"github.com/fengheasia/frp-warden/internal/security"
)

// SessionTTL 是 session 默认有效期(24 小时)。
const SessionTTL = 24 * time.Hour

// GenerateSessionToken 生成一个高熵的 session token(32 字节,hex 编码 64 字符)。
// token 明文只用于 cookie,数据库只保存其 hash。
func GenerateSessionToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("store: 生成 session token 失败: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// HashSessionToken 计算 session token 的 SHA-256 哈希。
// session token 是高熵随机值,SHA-256 足够(与密码的 bcrypt 区分:密码低熵需 bcrypt)。
func HashSessionToken(token string) string {
	return security.HashSHA256(token)
}

// CreateSession 创建一个 session,返回 session id。
// tokenHash 是 token 的哈希(绝不明文);expiresAt 是过期时间。
func (s *Store) CreateSession(ctx context.Context, adminID int64, tokenHash string, expiresAt time.Time) (string, error) {
	id, err := GenerateSessionToken()
	if err != nil {
		return "", err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, admin_id, token_hash, expires_at) VALUES (?, ?, ?, ?)`,
		id, adminID, tokenHash, expiresAt)
	if err != nil {
		return "", err
	}
	return id, nil
}

// GetSessionByTokenHash 按 token 哈希查询有效 session(未过期、未撤销)。
// 不存在或已失效返回 ErrSessionNotFound。
func (s *Store) GetSessionByTokenHash(ctx context.Context, tokenHash string) (model.Session, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, admin_id, token_hash, expires_at, created_at, revoked_at
		 FROM sessions WHERE token_hash = ? AND revoked_at IS NULL AND expires_at > CURRENT_TIMESTAMP`,
		tokenHash)
	var sess model.Session
	var revokedAt sql.NullTime
	err := row.Scan(&sess.ID, &sess.AdminID, &sess.TokenHash, &sess.ExpiresAt, &sess.CreatedAt, &revokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Session{}, ErrSessionNotFound
	}
	if err != nil {
		return model.Session{}, err
	}
	if revokedAt.Valid {
		sess.RevokedAt = &revokedAt.Time
	}
	return sess, nil
}

// RevokeSession 撤销指定 session(设置 revoked_at)。
// 不存在返回 ErrSessionNotFound。
func (s *Store) RevokeSession(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET revoked_at = CURRENT_TIMESTAMP WHERE id = ? AND revoked_at IS NULL`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrSessionNotFound
	}
	return nil
}

// RevokeAllSessionsByAdminID 撤销某管理员的所有有效 session(用于密码修改后全量撤销)。
func (s *Store) RevokeAllSessionsByAdminID(ctx context.Context, adminID int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET revoked_at = CURRENT_TIMESTAMP WHERE admin_id = ? AND revoked_at IS NULL`, adminID)
	return err
}
