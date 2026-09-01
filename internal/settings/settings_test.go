package settings

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"wedding-invitation/internal/config"
	"wedding-invitation/internal/models"
)

// newTestStore 构造一个使用内存 SQLite 的 Store，可选预置种子数据
func newTestStore(t *testing.T, seed map[string]string) *Store {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Setting{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	for k, v := range seed {
		if err := db.Create(&models.Setting{Key: k, Value: v}).Error; err != nil {
			t.Fatalf("写入种子失败: %v", err)
		}
	}
	return New(db, config.WeddingConfig{
		GroomName:    "李祥",
		BrideName:    "王羚羚",
		WeddingDate:  "2026-10-03",
		WeddingVenue: "麓湖·云栖草坪",
	})
}

func TestDefaults(t *testing.T) {
	s := newTestStore(t, nil)
	all := s.All()
	cases := map[string]string{
		KeyGroomName:        "李祥",
		KeyBrideName:        "王羚羚",
		KeyWeddingDate:      "2026-10-03",
		KeyVenue:            "麓湖·云栖草坪",
		KeyCardTransparency: "15",
		KeyGlassEnabled:     "true",
		KeyGlassBlur:        "16",
		KeyGlassSaturate:    "180",
		KeyCardColor:        "#f7f2eb",
	}
	for key, want := range cases {
		if got := all[key]; got != want {
			t.Errorf("默认 %s = %q, 期望 %q", key, got, want)
		}
	}
}

func TestDBOverridesDefault(t *testing.T) {
	s := newTestStore(t, map[string]string{
		KeyGroomName: "林予",
		KeyCardColor: "#ffffff",
	})
	all := s.All()
	if got := all[KeyGroomName]; got != "林予" {
		t.Errorf("DB 覆盖后的新郎名 = %q, 期望 %q", got, "林予")
	}
	if got := all[KeyCardColor]; got != "#ffffff" {
		t.Errorf("DB 覆盖后的颜色 = %q, 期望 %q", got, "#ffffff")
	}
	// 未覆盖的键仍返回默认值
	if got := all[KeyBrideName]; got != "王羚羚" {
		t.Errorf("未覆盖的新娘名 = %q, 期望 %q", got, "王羚羚")
	}
}

func TestSetAndGet(t *testing.T) {
	s := newTestStore(t, nil)
	if err := s.Set(KeyDateSub, "星期六 · 下午五时"); err != nil {
		t.Fatalf("Set 失败: %v", err)
	}
	if got := s.Get(KeyDateSub); got != "星期六 · 下午五时" {
		t.Errorf("Get = %q, 期望 %q", got, "星期六 · 下午五时")
	}
	// 已保存的值从新 Store 也应读到（持久化）
	s2 := New(s.db, config.WeddingConfig{BrideName: "王羚羚"})
	if got := s2.Get(KeyDateSub); got != "星期六 · 下午五时" {
		t.Errorf("重启后 Get = %q, 期望 %q", got, "星期六 · 下午五时")
	}
}

func TestSetUnknownKey(t *testing.T) {
	s := newTestStore(t, nil)
	if err := s.Set("not_a_key", "x"); err == nil {
		t.Fatal("设置未知键应报错")
	}
}

func TestSetInvalidValue(t *testing.T) {
	s := newTestStore(t, nil)
	invalid := []struct{ key, val string }{
		{KeyCardTransparency, "abc"},
		{KeyCardTransparency, "-1"},
		{KeyCardTransparency, "101"},
		{KeyGlassBlur, "60"},
		{KeyGlassBlur, "-5"},
		{KeyGlassSaturate, "50"},
		{KeyGlassSaturate, "301"},
		{KeyGlassEnabled, "yes"},
		{KeyCardColor, "red"},
		{KeyCardColor, "#12345"},
	}
	for _, c := range invalid {
		if err := s.Set(c.key, c.val); err == nil {
			t.Errorf("Set(%s=%q) 应报错", c.key, c.val)
		}
	}

	valid := []struct{ key, val string }{
		{KeyCardTransparency, "0"},
		{KeyCardTransparency, "100"},
		{KeyGlassBlur, "0"},
		{KeyGlassBlur, "50"},
		{KeyGlassSaturate, "100"},
		{KeyGlassSaturate, "300"},
		{KeyGlassEnabled, "true"},
		{KeyGlassEnabled, "false"},
		{KeyCardColor, "#abc"},
		{KeyCardColor, "#AABBCC"},
	}
	for _, c := range valid {
		if err := s.Set(c.key, c.val); err != nil {
			t.Errorf("Set(%s=%q) 不应报错: %v", c.key, c.val, err)
		}
	}
}

func TestTimeFields(t *testing.T) {
	s := newTestStore(t, nil)
	// 默认空（未配置时不展示）
	if got := s.Get(KeyCeremonyTime); got != "" {
		t.Errorf("默认仪式时间 = %q, 期望空", got)
	}
	if got := s.Get(KeyDinnerTime); got != "" {
		t.Errorf("默认晚宴时间 = %q, 期望空", got)
	}
	// 设置合法值
	if err := s.Set(KeyCeremonyTime, "15:30"); err != nil {
		t.Errorf("Set(ceremony_time) 报错: %v", err)
	}
	if err := s.Set(KeyDinnerTime, "18:00"); err != nil {
		t.Errorf("Set(dinner_time) 报错: %v", err)
	}
	if got := s.Get(KeyCeremonyTime); got != "15:30" {
		t.Errorf("仪式时间 = %q, 期望 %q", got, "15:30")
	}
	// 超长应报错
	if err := s.Set(KeyCeremonyTime, strings.Repeat("1", 21)); err == nil {
		t.Error("超长时间应报错")
	}
}

func TestMapLinkAndMusicURL(t *testing.T) {
	s := newTestStore(t, nil)
	// 默认值
	if got := s.Get(KeyMapLink); got != "" {
		t.Errorf("默认导航链接 = %q, 期望空", got)
	}
	if got := s.Get(KeyMusicURL); got != "/static/music/bgm.mp3" {
		t.Errorf("默认音乐地址 = %q, 期望 /static/music/bgm.mp3", got)
	}
	// 设置与回读
	if err := s.Set(KeyMapLink, "https://uri.amap.com/search?keyword=test"); err != nil {
		t.Errorf("Set(map_link) 报错: %v", err)
	}
	if err := s.Set(KeyMusicURL, "https://example.com/bgm.mp3"); err != nil {
		t.Errorf("Set(music_url) 报错: %v", err)
	}
	if got := s.Get(KeyMapLink); got != "https://uri.amap.com/search?keyword=test" {
		t.Errorf("导航链接 = %q", got)
	}
	// 超长应报错
	if err := s.Set(KeyMusicURL, strings.Repeat("a", 501)); err == nil {
		t.Error("超长 music_url 应报错")
	}
}

func TestValidateURLLinks(t *testing.T) {
	s := newTestStore(t, nil)
	valid := []struct{ key, val string }{
		{KeyMapLink, ""},
		{KeyMapLink, "https://uri.amap.com/search?keyword=x"},
		{KeyMapLink, "http://example.com/x"},
		{KeyMusicURL, "/static/music/bgm.mp3"},
		{KeyMusicURL, "https://cdn.example.com/bgm.mp3"},
	}
	for _, c := range valid {
		if err := s.Set(c.key, c.val); err != nil {
			t.Errorf("Set(%s=%q) 不应报错: %v", c.key, c.val, err)
		}
	}
	invalid := []struct{ key, val string }{
		{KeyMapLink, "javascript:alert(1)"},
		{KeyMapLink, "data:text/html,<script>alert(1)</script>"},
		{KeyMapLink, "vbscript:msgbox(1)"},
		{KeyMusicURL, "javascript:void(0)"},
	}
	for _, c := range invalid {
		if err := s.Set(c.key, c.val); err == nil {
			t.Errorf("Set(%s=%q) 应报错", c.key, c.val)
		}
	}
}

func TestValidate(t *testing.T) {
	if err := Validate(KeyGlassBlur, "20"); err != nil {
		t.Errorf("Validate 合法值应通过: %v", err)
	}
	if err := Validate(KeyGlassBlur, "x"); err == nil {
		t.Error("Validate 非法值应报错")
	}
	if err := Validate("unknown_key", "x"); err == nil {
		t.Error("Validate 未知键应报错")
	}
}
