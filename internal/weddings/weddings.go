package weddings

import (
	"crypto/rand"
	"regexp"
)

const tokenLen = 10

// tokenRe 校验 token：恰好 10 个随机小写字母（大小写无歧义，便于分享）
var tokenRe = regexp.MustCompile(`^[a-z]{10}$`)

// GenerateToken 生成 CSPRNG 随机 token：10 个小写字母（无取模偏差）
func GenerateToken() (string, error) {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	// 接受 [0, 234) 的字节，消除取模偏差（256 = 26*9 + 22）
	const maxByte = 26 * (256 / 26)
	out := make([]byte, tokenLen)
	buf := make([]byte, tokenLen*2)
	for i := 0; i < tokenLen; {
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		for _, b := range buf {
			if i >= tokenLen {
				break
			}
			if int(b) < maxByte {
				out[i] = letters[int(b)%26]
				i++
			}
		}
	}
	return string(out), nil
}

// IsValidToken 校验 token 格式（恰好 10 个小写字母）
func IsValidToken(tok string) bool {
	return tokenRe.MatchString(tok)
}
