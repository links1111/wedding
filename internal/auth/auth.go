package auth

import (
	"errors"
	"time"

	"wedding-invitation/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("用户名或密码错误")
)

// EnsureAdminUser 确保管理员用户存在，不存在则创建；已存在但密码不匹配则更新
func EnsureAdminUser(db *gorm.DB, username, password string) error {
	var admin models.AdminUser
	err := db.Where("username = ?", username).First(&admin).Error

	if err == gorm.ErrRecordNotFound {
		// 用户不存在，创建
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		admin = models.AdminUser{
			Username:     username,
			PasswordHash: string(hash),
		}
		return db.Create(&admin).Error
	}
	if err != nil {
		return err
	}

	// 用户已存在，检查密码是否匹配；不匹配则更新（支持通过环境变量修改密码）
	if bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)) != nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		return db.Model(&admin).Update("password_hash", string(hash)).Error
	}

	return nil
}

// VerifyAdmin 验证管理员凭据
func VerifyAdmin(db *gorm.DB, username, password string) (*models.AdminUser, error) {
	var admin models.AdminUser
	err := db.Where("username = ?", username).First(&admin).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return &admin, nil
}

// SessionToken 会话 token 信息
type SessionToken struct {
	Token     string
	ExpiresAt time.Time
}

// TokenStore 内存会话存储（单实例足够）
type TokenStore struct {
	tokens map[string]time.Time
}

// NewTokenStore 创建会话存储
func NewTokenStore() *TokenStore {
	return &TokenStore{tokens: make(map[string]time.Time)}
}

// Create 创建会话
func (s *TokenStore) Create(token string, ttl time.Duration) {
	s.tokens[token] = time.Now().Add(ttl)
}

// Valid 检查会话是否有效
func (s *TokenStore) Valid(token string) bool {
	exp, ok := s.tokens[token]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(s.tokens, token)
		return false
	}
	return true
}

// Destroy 销毁会话
func (s *TokenStore) Destroy(token string) {
	delete(s.tokens, token)
}

// Cleanup 清理过期会话
func (s *TokenStore) Cleanup() {
	now := time.Now()
	for k, exp := range s.tokens {
		if now.After(exp) {
			delete(s.tokens, k)
		}
	}
}
