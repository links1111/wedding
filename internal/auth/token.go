package auth

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateToken 生成 CSPRNG 随机会话 token
func GenerateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
