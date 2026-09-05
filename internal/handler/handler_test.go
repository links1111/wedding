package handler

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"wedding-invitation/internal/models"
	"wedding-invitation/internal/settings"
)

func TestValidateGuestInput(t *testing.T) {
	valid := []guestInput{
		{Name: "张三", Phone: "13800000000", Attending: 1, Headcount: 2, Message: "你好"},
		{Name: "  李四  ", Phone: "", Attending: 0, Headcount: 0, Message: ""},
		{Name: "王五", Phone: "123", Attending: 2, Headcount: 20, Message: strings.Repeat("x", 500)},
	}
	for i, in := range valid {
		if err := validateGuestInput(&in); err != nil {
			t.Errorf("用例%d 应通过: %v", i, err)
		}
	}

	// 副作用：trim 姓名、人数 0→1、超长留言截断
	v := guestInput{Name: "  李四  ", Headcount: 0, Message: strings.Repeat("x", 600)}
	if err := validateGuestInput(&v); err != nil {
		t.Fatalf("合法输入报错: %v", err)
	}
	if v.Name != "李四" {
		t.Errorf("姓名未 trim: %q", v.Name)
	}
	if v.Headcount != 1 {
		t.Errorf("人数未默认 1: %d", v.Headcount)
	}
	if len(v.Message) != 500 {
		t.Errorf("留言未截断为 500: %d", len(v.Message))
	}

	invalid := []guestInput{
		{Name: "", Phone: "", Attending: 1, Headcount: 1},
		{Name: strings.Repeat("张", 51), Phone: "", Attending: 1, Headcount: 1},
		{Name: "张三", Phone: strings.Repeat("1", 21), Attending: 1, Headcount: 1},
		{Name: "张三", Phone: "", Attending: 3, Headcount: 1},
		{Name: "张三", Phone: "", Attending: -1, Headcount: 1},
	}
	for i, in := range invalid {
		if err := validateGuestInput(&in); err == nil {
			t.Errorf("用例%d 应报错", i)
		}
	}
}

// newHandlerDB 构造已迁移的内存 SQLite 数据库
func newHandlerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&models.Wedding{}, &models.Setting{}, &models.Guest{}, &models.Visit{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	return db
}

// TestWeddingsTokenSeeding 验证 createWedding 创建婚礼并 seed 预填设置与 token 化音乐地址
func TestWeddingsTokenSeeding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newHandlerDB(t)
	h := New(db, nil, t.TempDir())

	r := gin.New()
	r.POST("/test", h.createWedding)
	req := httptest.NewRequest("POST", "/test",
		strings.NewReader(`{"name":"测试婚礼","groom_name":"张三","bride_name":"李四","wedding_date":"2026-10-03","venue":"成都"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, 期望 200, body=%s", rec.Code, rec.Body.String())
	}

	var w models.Wedding
	if err := db.First(&w).Error; err != nil {
		t.Fatalf("未创建婚礼: %v", err)
	}
	if w.Name != "测试婚礼" {
		t.Errorf("婚礼名 = %q, 期望 %q", w.Name, "测试婚礼")
	}
	if w.Token == "" {
		t.Fatal("token 为空")
	}

	all := h.settingsFor(w.ID).All()
	if got := all[settings.KeyGroomName]; got != "张三" {
		t.Errorf("groom_name = %q, 期望 %q", got, "张三")
	}
	if got := all[settings.KeyBrideName]; got != "李四" {
		t.Errorf("bride_name = %q, 期望 %q", got, "李四")
	}
	wantMusic := "/static/" + w.Token + "/music/bgm.mp3"
	if got := all[settings.KeyMusicURL]; got != wantMusic {
		t.Errorf("music_url = %q, 期望 %q", got, wantMusic)
	}
}

// TestDeleteWeddingCascade 验证 deleteWedding 事务级联删除关联数据
func TestDeleteWeddingCascade(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newHandlerDB(t)
	h := New(db, nil, t.TempDir())

	w := models.Wedding{Token: "tok_delete_test_1234567890", Name: "待删除"}
	if err := db.Create(&w).Error; err != nil {
		t.Fatalf("创建婚礼失败: %v", err)
	}
	for _, row := range []interface{}{
		&models.Setting{WeddingID: w.ID, Key: "venue", Value: "成都"},
		&models.Guest{WeddingID: w.ID, Name: "张三"},
		&models.Visit{WeddingID: w.ID, IP: "1.2.3.4"},
	} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("创建关联数据失败: %v", err)
		}
	}

	r := gin.New()
	r.DELETE("/test/:id", h.deleteWedding)
	req := httptest.NewRequest("DELETE", "/test/"+strconv.FormatInt(w.ID, 10), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, 期望 200, body=%s", rec.Code, rec.Body.String())
	}

	var count int64
	if err := db.Model(&models.Wedding{}).Where("id = ?", w.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("婚礼未删除: 剩余 %d", count)
	}
	for _, m := range []interface{}{&models.Setting{}, &models.Guest{}, &models.Visit{}} {
		if err := db.Model(m).Where("wedding_id = ?", w.ID).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Errorf("%T 未级联删除: 剩余 %d", m, count)
		}
	}
}
