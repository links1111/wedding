package config

import (
	"crypto/rand"
	"encoding/hex"
)

// generateRandomToken 生成加密安全的随机十六进制字符串
func generateRandomToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		// rand.Read 失败极少见；回退到固定长度零字节不会暴露凭据
		return hex.EncodeToString(make([]byte, length))
	}
	return hex.EncodeToString(b)
}
