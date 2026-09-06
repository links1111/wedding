package weddings

import (
	"regexp"
	"testing"
)

func TestGenerateTokenLengthAndCharset(t *testing.T) {
	tok, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken err: %v", err)
	}
	if len(tok) != 10 {
		t.Errorf("token 长度 = %d, 期望 10", len(tok))
	}
	if !IsValidToken(tok) {
		t.Errorf("生成的 token 未通过校验: %q", tok)
	}
	if !regexp.MustCompile(`^[a-z]+$`).MatchString(tok) {
		t.Errorf("token 应全为小写字母: %q", tok)
	}
	// 唯一性抽样
	a, _ := GenerateToken()
	b, _ := GenerateToken()
	if a == b {
		t.Error("两次生成相同 token")
	}
}

func TestIsValidTokenRejectsBad(t *testing.T) {
	bad := []string{
		"",
		"abc",            // 太短
		"abcdefghijklmn", // 太长
		"Abcdefghij",     // 含大写
		"abcd3fghij",     // 含数字
		"abc defghij",    // 含空格
		"../abcdefg",     // 路径穿越字符
		"a;drop table",
		"abcdefgh_", // 含下划线
	}
	for _, s := range bad {
		if IsValidToken(s) {
			t.Errorf("应拒绝: %q", s)
		}
	}
	good := []string{"abcdefghij", "zzzzzzzzzz", "zkyqmtwvba"}
	for _, s := range good {
		if !IsValidToken(s) {
			t.Errorf("应通过: %q", s)
		}
	}
}
