// Package security 封装与密码/密钥相关的安全原语:强随机密码生成、bcrypt 哈希与校验、
// SHA-256 哈希(用于高熵 session token)。
//
// 设计要点(见 docs/DECISIONS/0008-admin-password-hashing.md 与 0011):
//   - 密码只以 bcrypt 哈希存储,绝不保存明文。
//   - bcrypt cost 使用 bcrypt.DefaultCost 这一合理默认值。
//   - session token 是高熵随机值,SHA-256 足够(与密码的 bcrypt 区分:密码低熵需 bcrypt)。
//   - 接口保持简单,为未来切换到 argon2id 预留空间,但本轮不过度设计。
package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

// passwordAlphabet 是生成随机密码使用的字符集。
// 刻意剔除容易混淆的字符(0/O、1/l/I 等),便于人工复制与转录。
const passwordAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"

// defaultPasswordLength 是默认生成的密码长度(落在建议的 16-24 区间内)。
const defaultPasswordLength = 20

// GenerateRandomPassword 使用 crypto/rand 生成长度为 n 的强随机密码。
// 若 n <= 0,则使用默认长度。绝不使用 math/rand。
func GenerateRandomPassword(n int) (string, error) {
	if n <= 0 {
		n = defaultPasswordLength
	}
	max := big.NewInt(int64(len(passwordAlphabet)))
	buf := make([]byte, n)
	for i := range buf {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("security: 生成随机密码失败: %w", err)
		}
		buf[i] = passwordAlphabet[idx.Int64()]
	}
	return string(buf), nil
}

// HashPassword 使用 bcrypt(DefaultCost)对明文密码进行哈希。
func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("security: 计算密码哈希失败: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword 校验明文密码是否与给定 bcrypt 哈希匹配。
func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// HashSecret 对任意密钥(如 tenant token)计算 bcrypt 哈希。
// 与 HashPassword 同实现,仅在语义上区分用途:数据库只保存该 hash,绝不保存明文。
func HashSecret(plain string) (string, error) {
	return HashPassword(plain)
}

// VerifySecret 以常量时间方式校验明文密钥与 bcrypt 哈希是否匹配(用于 tenant token 校验)。
// 绝不直接比较明文,也不支持明文 token_hash 兼容模式。
func VerifySecret(hash, plain string) bool {
	return VerifyPassword(hash, plain)
}

// HashSHA256 计算字符串的 SHA-256 哈希并返回 hex 编码。
// 用于高熵 session token 的哈希存储(与密码的 bcrypt 区分:session token 是高熵随机值,
// SHA-256 足够且比 bcrypt 快得多;密码低熵必须用 bcrypt)。
func HashSHA256(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
