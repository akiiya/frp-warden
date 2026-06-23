package store

import (
	"context"
	"strings"

	"github.com/fengheasia/frp-warden/internal/model"
)

// InsertAuditLog 写入一条审计日志。message 中文为主,绝不包含密码/token/hash。
func (s *Store) InsertAuditLog(ctx context.Context, entry model.AuditLog) error {
	if strings.TrimSpace(entry.ActorType) == "" || strings.TrimSpace(entry.Action) == "" {
		return ErrEmptyField
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO audit_logs (actor_type, actor_id, action, target_type, target_id, message, ip)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entry.ActorType, entry.ActorID, entry.Action, entry.TargetType, entry.TargetID, entry.Message, entry.IP)
	return err
}

// ListAuditLogs 返回最近 n 条审计日志(按 created_at 降序)。n <= 0 时默认 100。
func (s *Store) ListAuditLogs(ctx context.Context, n int) ([]model.AuditLog, error) {
	if n <= 0 {
		n = 100
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, actor_type, actor_id, action, target_type, target_id, message, ip, created_at
		 FROM audit_logs ORDER BY id DESC LIMIT ?`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.AuditLog
	for rows.Next() {
		var e model.AuditLog
		if err := rows.Scan(&e.ID, &e.ActorType, &e.ActorID, &e.Action,
			&e.TargetType, &e.TargetID, &e.Message, &e.IP, &e.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}
