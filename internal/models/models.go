package models

import "time"

// Visit 访问记录
type Visit struct {
	ID          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
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
