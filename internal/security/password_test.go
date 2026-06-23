package security

import (
	"strings"
	"testing"
)

func TestGenerateRandomPassword(t *testing.T) {
	pw, err := GenerateRandomPassword(20)
	if err != nil {
		t.Fatalf("生成密码失败: %v", err)
	}
	if len(pw) != 20 {
		t.Errorf("密码长度 = %d, 期望 20", len(pw))
	}
	// 不应包含易混淆字符。
	for _, c := range "0O1lI" {
		if strings.ContainsRune(pw, c) {
			t.Errorf("密码不应包含易混淆字符 %q: %s", c, pw)
		}
	}
	// 两次生成应不同(极大概率)。
	pw2, _ := GenerateRandomPassword(20)
	if pw == pw2 {
		t.Error("两次随机生成的密码不应相同")
	}
}

func TestGenerateRandomPasswordDefaultLength(t *testing.T) {
	pw, err := GenerateRandomPassword(0)
	if err != nil {
		t.Fatalf("生成密码失败: %v", err)
	}
	if len(pw) != defaultPasswordLength {
		t.Errorf("默认长度密码 = %d, 期望 %d", len(pw), defaultPasswordLength)
	}
}

func TestHashAndVerify(t *testing.T) {
	plain := "S3cretPassw0rdXYZ"
	hash, err := HashPassword(plain)
	if err != nil {
		t.Fatalf("哈希失败: %v", err)
	}
	// 哈希不能等于明文。
	if hash == plain {
		t.Error("password_hash 不应等于明文密码")
	}
	if strings.Contains(hash, plain) {
		t.Error("password_hash 不应包含明文密码")
	}
	// 正确密码校验通过,错误密码校验失败。
	if !VerifyPassword(hash, plain) {
		t.Error("正确密码应校验通过")
	}
	if VerifyPassword(hash, "wrong-password") {
		t.Error("错误密码不应校验通过")
	}
}
