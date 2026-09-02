package main

import (
	"context"
	"embed"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/natefinch/lumberjack.v2"

	"wedding-invitation/internal/auth"
	"wedding-invitation/internal/config"
	"wedding-invitation/internal/db"
	"wedding-invitation/internal/handler"
	"wedding-invitation/internal/settings"
)

func main() {
	cfg := config.Load()

	// 初始化日志（文件轮转 + 控制台）
	initLogger(cfg)

	// 生产环境关闭 Gin 调试日志
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 初始化数据库
	if err := db.InitDB(cfg.Database.Path); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer db.CloseDB()

	// 确保管理员存在
	if err := auth.EnsureAdminUser(db.DB, cfg.Admin.User, cfg.Admin.Pass); err != nil {
		log.Fatalf("创建管理员失败: %v", err)
	}

	// 协议：配置了 TLS 证书且文件存在则 HTTPS，否则 HTTP
	scheme := "http"
	if cfg.TLS.CertFile != "" && cfg.TLS.KeyFile != "" {
		if _, err := os.Stat(cfg.TLS.CertFile); err == nil {
			scheme = "https"
		} else {
			log.Printf("TLS 证书文件不存在 (%s)，回退 HTTP", cfg.TLS.CertFile)
		}
	}

	// 打印管理员凭据（仅首次）
	if isFreshDB() {
		log.Printf("========================================")
		log.Printf("管理员账号: %s", cfg.Admin.User)
		log.Printf("管理员密码: %s (请尽快修改)", cfg.Admin.Pass)
		log.Printf("管理后台: %s://localhost:%s/admin", scheme, cfg.Server.Port)
		log.Printf("========================================")
	}

	// 会话存储
	sessions := auth.NewTokenStore()
	go cleanupSessions(sessions)

	// Gin 引擎
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(securityHeaders())

	// API 处理器
	settingsStore := settings.New(db.DB, cfg.Wedding)
	h := handler.New(db.DB, sessions, settingsStore, cfg.Paths.StaticDir)
	h.RegisterRoutes(r)

	// 静态资源（图片等），从 STATIC_DIR 提供，Docker 可挂载映射
	if _, err := os.Stat(cfg.Paths.StaticDir); err == nil {
		r.Static("/static", cfg.Paths.StaticDir)
		log.Printf("静态资源目录: %s -> /static", cfg.Paths.StaticDir)
	} else {
		log.Printf("静态资源目录不存在，跳过: %s", cfg.Paths.StaticDir)
	}

	// HTML 模板：优先从 TEMPLATE_DIR 加载（Docker 可挂载），否则用嵌入的 embed.FS
	if _, err := os.Stat(cfg.Paths.TemplateDir); err == nil {
		r.LoadHTMLGlob(cfg.Paths.TemplateDir + "/*")
		log.Printf("模板目录(本地): %s", cfg.Paths.TemplateDir)
		r.GET("/", func(c *gin.Context) {
			c.HTML(http.StatusOK, "index.html", nil)
		})
		r.GET("/admin", func(c *gin.Context) {
			c.HTML(http.StatusOK, "admin.html", nil)
		})
	} else {
		loadEmbeddedTemplates(r)
		log.Printf("模板目录(嵌入): web/templates")
	}

	// 监听地址：server.host 未配置则监听所有网卡
	bindHost := cfg.Server.Host
	if bindHost == "" {
		bindHost = "0.0.0.0"
	}
	server := &http.Server{
		Addr:         net.JoinHostPort(bindHost, cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 优雅关闭
	go func() {
		log.Printf("服务器启动: %s://localhost:%s", scheme, cfg.Server.Port)
		log.Printf("请柬页面: %s://localhost:%s/", scheme, cfg.Server.Port)
		log.Printf("管理后台: %s://localhost:%s/admin", scheme, cfg.Server.Port)
		if err := serve(server, scheme, cfg.TLS.CertFile, cfg.TLS.KeyFile); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭服务器...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("服务器关闭错误: %v", err)
	}
	log.Println("服务器已关闭")
}

// initLogger 初始化日志输出：同时写入文件（lumberjack 轮转）和控制台
func initLogger(cfg *config.Config) {
	var writers []io.Writer

	// 控制台输出
	writers = append(writers, os.Stdout)

	// 文件输出
	if cfg.Log.File != "" {
		// 确保日志目录存在
		if dir := filepath.Dir(cfg.Log.File); dir != "" && dir != "." {
			_ = os.MkdirAll(dir, 0755)
		}
		lj := &lumberjack.Logger{
			Filename:   cfg.Log.File,
			MaxSize:    cfg.Log.MaxSize,
			MaxBackups: cfg.Log.MaxBackups,
			MaxAge:     cfg.Log.MaxAge,
			Compress:   true,
		}
		writers = append(writers, lj)
		log.Printf("日志文件: %s (maxSize=%dMB, maxBackups=%d, maxAge=%ddays)", cfg.Log.File, cfg.Log.MaxSize, cfg.Log.MaxBackups, cfg.Log.MaxAge)
	}

	// 设置日志输出
	log.SetOutput(io.MultiWriter(writers...))

	// 根据级别设置日志前缀标志
	switch strings.ToLower(cfg.Log.Level) {
	case "debug":
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	default:
		log.SetFlags(log.Ldate | log.Ltime)
	}
}

// securityHeaders 安全响应头中间件
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}

// isFreshDB 检查是否为全新数据库（通过检查 data 目录是否刚创建）
func isFreshDB() bool {
	info, err := os.Stat("./data")
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) < 5*time.Second
}

// cleanupSessions 定期清理过期会话
func cleanupSessions(sessions *auth.TokenStore) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		sessions.Cleanup()
	}
}

// serve 按协议启动 HTTP 或 HTTPS 监听
func serve(server *http.Server, scheme, certFile, keyFile string) error {
	if scheme == "https" {
		return server.ListenAndServeTLS(certFile, keyFile)
	}
	return server.ListenAndServe()
}

// loadEmbeddedTemplates 从 embed.FS 加载模板（回退方案）
//
//go:embed web/templates
var templateFS embed.FS

func loadEmbeddedTemplates(r *gin.Engine) {
	tmpl, err := template.New("").ParseFS(templateFS, "web/templates/*.html")
	if err != nil {
		log.Fatalf("嵌入模板加载失败: %v", err)
	}
	r.SetHTMLTemplate(tmpl)
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})
	r.GET("/admin", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin.html", nil)
	})
}
