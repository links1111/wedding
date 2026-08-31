package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Admin    AdminConfig    `yaml:"admin"`
	Wedding  WeddingConfig  `yaml:"wedding"`
	Paths    PathsConfig    `yaml:"paths"`
	Log      LogConfig      `yaml:"log"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string `yaml:"port"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// AdminConfig 管理员配置
type AdminConfig struct {
	User      string `yaml:"user"`
	Pass      string `yaml:"pass"`
	JWTSecret string `yaml:"jwt_secret"`
}

// WeddingConfig 婚礼信息配置
type WeddingConfig struct {
	GroomName    string `yaml:"groom_name"`
	BrideName    string `yaml:"bride_name"`
	WeddingDate  string `yaml:"wedding_date"`
	WeddingVenue string `yaml:"wedding_venue"`
}

// PathsConfig 路径配置
type PathsConfig struct {
	StaticDir   string `yaml:"static_dir"`
	TemplateDir string `yaml:"template_dir"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `yaml:"level"`
	File       string `yaml:"file"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
}

// Load 从 config.yml 加载配置，环境变量可覆盖同名配置
func Load() *Config {
	cfg := &Config{
		Server:   ServerConfig{Port: "8080"},
		Database: DatabaseConfig{Path: "./data/wedding.db"},
		Admin:    AdminConfig{User: "admin"},
		Wedding: WeddingConfig{
			GroomName:    "新郎",
			BrideName:    "新娘",
			WeddingDate:  "2025-10-01",
			WeddingVenue: "婚礼殿堂",
		},
		Paths: PathsConfig{
			StaticDir:   "./web/static",
			TemplateDir: "./web/templates",
		},
		Log: LogConfig{
			Level:      "info",
			File:       "",
			MaxSize:    10,
			MaxBackups: 5,
			MaxAge:     30,
		},
	}

	// 尝试从 config.yml 加载
	configFile := getEnv("CONFIG_FILE", "config.yml")
	if data, err := os.ReadFile(configFile); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			fmt.Printf("解析配置文件 %s 失败: %v，使用默认值\n", configFile, err)
		}
	}

	// 环境变量覆盖（优先级最高）
	cfg.Server.Port = getEnv("PORT", cfg.Server.Port)
	cfg.Database.Path = getEnv("DB_PATH", cfg.Database.Path)
	cfg.Admin.User = getEnv("ADMIN_USER", cfg.Admin.User)
	cfg.Admin.Pass = getEnv("ADMIN_PASS", cfg.Admin.Pass)
	cfg.Admin.JWTSecret = getEnv("JWT_SECRET", cfg.Admin.JWTSecret)
	cfg.Wedding.GroomName = getEnv("GROOM_NAME", cfg.Wedding.GroomName)
	cfg.Wedding.BrideName = getEnv("BRIDE_NAME", cfg.Wedding.BrideName)
	cfg.Wedding.WeddingDate = getEnv("WEDDING_DATE", cfg.Wedding.WeddingDate)
	cfg.Wedding.WeddingVenue = getEnv("WEDDING_VENUE", cfg.Wedding.WeddingVenue)
	cfg.Paths.StaticDir = getEnv("STATIC_DIR", cfg.Paths.StaticDir)
	cfg.Paths.TemplateDir = getEnv("TEMPLATE_DIR", cfg.Paths.TemplateDir)
	cfg.Log.File = getEnv("LOG_FILE", cfg.Log.File)
	cfg.Log.Level = getEnv("LOG_LEVEL", cfg.Log.Level)

	// 如果未设置密码或密钥，生成随机值并提示
	if cfg.Admin.Pass == "" {
		cfg.Admin.Pass = generateRandomToken(16)
	}
	if cfg.Admin.JWTSecret == "" {
		cfg.Admin.JWTSecret = generateRandomToken(32)
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fallback
		}
		return n
	}
	return fallback
}
