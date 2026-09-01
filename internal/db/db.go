package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"wedding-invitation/internal/models"
)

// DB 全局 GORM 数据库实例
var DB *gorm.DB

// InitDB 初始化 SQLite 数据库（使用 GORM）并自动迁移表结构
func InitDB(dbPath string) error {
	// 确保目录存在
	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建数据库目录失败: %w", err)
		}
	}

	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on", dbPath)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return fmt.Errorf("打开数据库失败: %w", err)
	}

	// SQLite 写入并发限制
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取底层 SQL DB 失败: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	// 自动迁移表结构
	if err := db.AutoMigrate(&models.Visit{}, &models.Guest{}, &models.AdminUser{}, &models.Setting{}); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	// 创建索引（幂等）
	if err := createIndexes(db); err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}

	DB = db
	log.Println("数据库初始化成功 (GORM):", dbPath)
	return nil
}

func createIndexes(db *gorm.DB) error {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_visits_visited_at ON visits(visited_at)",
		"CREATE INDEX IF NOT EXISTS idx_guests_created_at ON guests(created_at)",
	}
	for _, sql := range indexes {
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
	}
	return nil
}

// CloseDB 关闭数据库
func CloseDB() {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}
}
