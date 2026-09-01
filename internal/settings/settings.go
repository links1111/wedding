package settings

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"wedding-invitation/internal/config"
	"wedding-invitation/internal/models"
)

// 设置键（管理后台可配置项）
const (
	// 请柬文案
	KeyGroomName     = "groom_name"
	KeyBrideName     = "bride_name"
	KeyWeddingDate   = "wedding_date"
	KeyDateMain      = "date_main"
	KeyDateSub       = "date_sub"
	KeyVenue         = "venue"
	KeyCeremonyTitle = "ceremony_title"
	KeyCeremonySub   = "ceremony_sub"
	KeyCeremonyTime  = "ceremony_time"
	KeyDinnerTitle   = "dinner_title"
	KeyDinnerSub     = "dinner_sub"
	KeyDinnerTime    = "dinner_time"
	KeyDressTitle    = "dress_title"
	KeyDressSub      = "dress_sub"
	KeyFooter1       = "footer1"
	KeyFooter2       = "footer2"
	KeyFooter3       = "footer3"
	KeyHandwrite     = "handwrite"
	KeyChineseTitle  = "chinese_title"

	// 卡片样式
	KeyCardTransparency = "card_transparency"
	KeyGlassEnabled     = "glass_enabled"
	KeyGlassBlur        = "glass_blur"
	KeyGlassSaturate    = "glass_saturate"
	KeyCardColor        = "card_color"
)

var (
	ErrUnknownKey   = errors.New("未知的设置键")
	ErrInvalidValue = errors.New("设置值不合法")
)

var hexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{3}([0-9a-fA-F]{3})?$`)

// textKeys 纯文本设置键 → 最大长度
var textKeys = map[string]int{
	KeyGroomName:     50,
	KeyBrideName:     50,
	KeyWeddingDate:   20,
	KeyDateMain:      100,
	KeyDateSub:       100,
	KeyVenue:         100,
	KeyCeremonyTitle: 50,
	KeyCeremonySub:   100,
	KeyCeremonyTime:  20,
	KeyDinnerTitle:   50,
	KeyDinnerSub:     100,
	KeyDinnerTime:    20,
	KeyDressTitle:    50,
	KeyDressSub:      100,
	KeyFooter1:       50,
	KeyFooter2:       50,
	KeyFooter3:       50,
	KeyHandwrite:     100,
	KeyChineseTitle:  50,
}

// rangeKey 数值区间设置键
type rangeKey struct{ min, max int }

var rangeKeys = map[string]rangeKey{
	KeyCardTransparency: {0, 100},
	KeyGlassBlur:        {0, 50},
	KeyGlassSaturate:    {100, 300},
}

// Store 读取/写入设置：默认值来自 config，DB 中的值覆盖默认值
type Store struct {
	db       *gorm.DB
	defaults map[string]string
}

// New 创建 Store，defaults 以 config 中的婚礼信息 + 代码常量为种子
func New(db *gorm.DB, wedding config.WeddingConfig) *Store {
	return &Store{db: db, defaults: defaultSettings(wedding)}
}

func defaultSettings(w config.WeddingConfig) map[string]string {
	return map[string]string{
		KeyGroomName:     w.GroomName,
		KeyBrideName:     w.BrideName,
		KeyWeddingDate:   w.WeddingDate,
		KeyVenue:         w.WeddingVenue,
		KeyDateMain:      "2026 · 十月 · 三日",
		KeyDateSub:       "星期日 · 傍晚六时",
		KeyCeremonyTitle: "仪式",
		KeyCeremonySub:   "暮光花园",
		KeyCeremonyTime:  "",
		KeyDinnerTitle:   "晚宴",
		KeyDinnerSub:     "水岸餐厅",
		KeyDinnerTime:    "",
		KeyDressTitle:    "着装",
		KeyDressSub:      "优雅正装",
		KeyFooter1:       "敬候光临",
		KeyFooter2:       "2026 · 秋",
		KeyFooter3:       "成都",
		KeyHandwrite:     "with love, always.",
		KeyChineseTitle:  "邀 请 函",

		KeyCardTransparency: "15",
		KeyGlassEnabled:     "true",
		KeyGlassBlur:        "16",
		KeyGlassSaturate:    "180",
		KeyCardColor:        "#f7f2eb",
	}
}

// All 返回合并后的全部设置（默认值 + DB 覆盖）
func (s *Store) All() map[string]string {
	out := make(map[string]string, len(s.defaults))
	for k, v := range s.defaults {
		out[k] = v
	}
	if s.db == nil {
		return out
	}
	var rows []models.Setting
	if err := s.db.Find(&rows).Error; err != nil {
		return out
	}
	for _, r := range rows {
		// 只接受已知键，忽略数据库中的陌生键
		if _, ok := s.defaults[r.Key]; ok {
			out[r.Key] = r.Value
		}
	}
	return out
}

// Get 返回单个设置值（无默认值时为空字符串）
func (s *Store) Get(key string) string {
	return s.All()[key]
}

// Set 校验并写入（upsert）单个设置
func (s *Store) Set(key, value string) error {
	if _, ok := s.defaults[key]; !ok {
		return ErrUnknownKey
	}
	if err := Validate(key, value); err != nil {
		return err
	}
	return s.db.Save(&models.Setting{Key: key, Value: value}).Error
}

// Validate 校验单个设置键值是否合法
func Validate(key, value string) error {
	if maxLen, ok := textKeys[key]; ok {
		if len(value) > maxLen {
			return fmt.Errorf("%w: %s 长度超过 %d", ErrInvalidValue, key, maxLen)
		}
		return nil
	}
	if r, ok := rangeKeys[key]; ok {
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || n < r.min || n > r.max {
			return fmt.Errorf("%w: %s 需为 %d-%d 的整数", ErrInvalidValue, key, r.min, r.max)
		}
		return nil
	}
	switch key {
	case KeyGlassEnabled:
		if value != "true" && value != "false" {
			return fmt.Errorf("%w: %s 需为 true 或 false", ErrInvalidValue, key)
		}
	case KeyCardColor:
		if !hexColorRe.MatchString(strings.TrimSpace(value)) {
			return fmt.Errorf("%w: %s 需为 #RGB 或 #RRGGBB", ErrInvalidValue, key)
		}
	default:
		return ErrUnknownKey
	}
	return nil
}
