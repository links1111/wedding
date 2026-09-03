package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTempConfig 写临时配置文件并返回路径
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadTLS(t *testing.T) {
	t.Setenv("CONFIG_FILE", writeTempConfig(t, `server:
  port: "8080"
tls:
  cert_file: "/etc/ssl/cert.pem"
  key_file: "/etc/ssl/key.pem"
  https_port: "8443"
`))
	t.Setenv("TLS_CERT_FILE", "")
	t.Setenv("TLS_KEY_FILE", "")
	t.Setenv("HTTPS_PORT", "")

	cfg := Load()
	if cfg.Server.Port != "8080" {
		t.Errorf("HTTP 端口 = %q, 期望 %q", cfg.Server.Port, "8080")
	}
	if cfg.TLS.HTTPSPort != "8443" {
		t.Errorf("HTTPS 端口 = %q, 期望 %q", cfg.TLS.HTTPSPort, "8443")
	}
	if cfg.TLS.CertFile != "/etc/ssl/cert.pem" {
		t.Errorf("证书路径 = %q, 期望 /etc/ssl/cert.pem", cfg.TLS.CertFile)
	}
	if cfg.TLS.KeyFile != "/etc/ssl/key.pem" {
		t.Errorf("私钥路径 = %q, 期望 /etc/ssl/key.pem", cfg.TLS.KeyFile)
	}
}

func TestLoadTLSServerLevel(t *testing.T) {
	// 兼容把路径写在 server 段下的场景（yaml 不关心层级由 struct 决定）
	t.Setenv("CONFIG_FILE", writeTempConfig(t, `server:
  port: "8080"
`))
	t.Setenv("TLS_CERT_FILE", "/tmp/custom.pem")
	t.Setenv("TLS_KEY_FILE", "/tmp/custom.key")

	cfg := Load()
	if cfg.TLS.CertFile != "/tmp/custom.pem" {
		t.Errorf("env 覆盖后证书路径 = %q", cfg.TLS.CertFile)
	}
	if cfg.TLS.KeyFile != "/tmp/custom.key" {
		t.Errorf("env 覆盖后私钥路径 = %q", cfg.TLS.KeyFile)
	}
}

func TestDefaultTLSDisabled(t *testing.T) {
	t.Setenv("CONFIG_FILE", writeTempConfig(t, "server:\n  port: \"8080\"\n"))
	t.Setenv("TLS_CERT_FILE", "")
	t.Setenv("TLS_KEY_FILE", "")

	cfg := Load()
	if cfg.TLS.CertFile != "" || cfg.TLS.KeyFile != "" {
		t.Errorf("未配置 tls 时应为空: %q %q", cfg.TLS.CertFile, cfg.TLS.KeyFile)
	}
}
