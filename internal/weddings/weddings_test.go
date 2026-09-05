package weddings

import (
	"strings"
	"testing"
)

func TestGenerateTokenLengthAndCharset(t *testing.T) {
	tok, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken err: %v", err)
	}
	if len(tok) < 40 { // 32 字节 → base64url 43 字符
		t.Errorf("token 过短: %d", len(tok))
	}
	if !IsValidToken(tok) {
		t.Errorf("生成的 token 未通过校验: %q", tok)
	}
	// 唯一性抽样
	a, _ := GenerateToken()
	b, _ := GenerateToken()
	if a == b {
		t.Error("两次生成相同 token")
	}
}

func TestIsValidTokenRejectsBad(t *testing.T) {
	bad := []string{"", "abc", "../x", "a b c", "a;drop table", strings.Repeat("a", 100)}
	for _, s := range bad {
		if IsValidToken(s) {
			t.Errorf("应拒绝: %q", s)
		}
	}
	good := []string{"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"}
	for _, s := range good {
		if !IsValidToken(s) {
			t.Errorf("应通过: %q", s)
		}
	}
}
