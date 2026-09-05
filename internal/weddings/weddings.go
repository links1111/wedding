package weddings

import (
	"crypto/rand"
	"encoding/base64"
	"regexp"
)

var tokenRe = regexp.MustCompile(`^[A-Za-z0-9_-]{20,64}$`)

// GenerateToken 生成 CSPRNG 婚礼 token：32 字节 → base64url
func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// IsValidToken 校验 token 格式（字母数字与 -_ 下划线，长度 20-64）
func IsValidToken(tok string) bool {
	return tokenRe.MatchString(tok)
}
