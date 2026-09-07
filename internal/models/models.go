package models

import "time"

// Wedding 一场婚礼（多租户），双 token：公开请柬 token + 私密后台 admin token
type Wedding struct {
	ID         int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Token      string    `json:"token" gorm:"uniqueIndex;not null;size:64"`  // 公开请柬 token
	AdminToken string    `json:"-" gorm:"uniqueIndex;not null;size:64"`      // 新人后台 token（仅系统总览可见，不随模型序列化）
	Name       string    `json:"name" gorm:"not null;size:100"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// 子卡片预设类型 key（无自定义）
const (
	CardTypeMeet      = "meet"      // 相遇
	CardTypeHeartbeat = "heartbeat" // 心动瞬间
	CardTypeDate      = "date"      // 约会
	CardTypeConfess   = "confess"   // 表白
	CardTypeTogether  = "together"  // 在一起
	CardTypeTravel    = "travel"    // 旅行
	CardTypePropose   = "propose"   // 求婚
	CardTypeEngage    = "engage"    // 订婚
)

// Card 子卡片（我们的故事）：主请柬由 settings 驱动，卡片表只存子卡片
type Card struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	WeddingID int64     `json:"-" gorm:"index;not null"`
	Sort      int       `json:"sort" gorm:"not null;default:0"`
	Type      string    `json:"type" gorm:"size:32;not null;default:'custom'"`
	Title     string    `json:"title" gorm:"size:100;not null;default:''"`
	Date      string    `json:"date" gorm:"size:50;not null;default:''"`
	Content   string    `json:"content" gorm:"size:2000;not null;default:''"`
	Images    string    `json:"images" gorm:"size:2000;not null;default:''"` // 本卡背景图文件名，逗号分隔
	Enabled   bool      `json:"enabled" gorm:"not null"`                     // 不设 DB default，避免 GORM 省略 false
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// Visit 访问记录
type Visit struct {
	ID          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	WeddingID   int64     `json:"wedding_id" gorm:"index;not null;default:0"`
	IP          string    `json:"ip" gorm:"not null"`
	UserAgent   string    `json:"user_agent" gorm:"default:''"`
	Referer     string    `json:"referer" gorm:"default:''"`
	VisitorName string    `json:"visitor_name" gorm:"default:''"`
	VisitedAt   time.Time `json:"visited_at" gorm:"autoCreateTime"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// Guest 来宾 RSVP 回复
type Guest struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	WeddingID int64     `json:"wedding_id" gorm:"index;not null;default:0"`
	Name      string    `json:"name" gorm:"not null"`
	Phone     string    `json:"phone" gorm:"default:''"`
	Attending int       `json:"attending" gorm:"default:0"` // 0=未确认 1=出席 2=缺席
	Headcount int       `json:"headcount" gorm:"default:1"`
	Message   string    `json:"message" gorm:"default:''"`
	IP        string    `json:"ip" gorm:"default:''"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// AdminUser 管理员
type AdminUser struct {
	ID           int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Username     string    `json:"username" gorm:"uniqueIndex;not null"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// Setting 键值对设置，按婚礼隔离
type Setting struct {
	WeddingID int64  `json:"wedding_id" gorm:"primaryKey"`
	Key       string `json:"key" gorm:"primaryKey"`
	Value     string `json:"value" gorm:"not null"`
}

// Attending 状态常量
const (
	AttendingUnknown = 0
	AttendingYes     = 1
	AttendingNo      = 2
)

// AttendingLabel 中文标签
func AttendingLabel(a int) string {
	switch a {
	case AttendingYes:
		return "出席"
	case AttendingNo:
		return "缺席"
	default:
		return "未确认"
	}
}
