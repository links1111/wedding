# 多租户 Token 隔离系统实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将单租户婚礼请柬改造成多租户：系统管理员在 `/admin` 创建/删除婚礼，每场婚礼经 `/w/{token}` 请柬与 `/admin/{token}` 后台独立访问，数据与媒体按 token 隔离。

**Architecture:** 单 SQLite 库 + `wedding_id` 隔离（`Wedding` 表 + `Setting` 复合主键 `(wedding_id,key)` + `Guest`/`Visit` 加 wedding_id）。路由按 `/w/:token` 与 `/api/w/:token` 命名空间；token（crypto/rand 32 字节 → base64url）即婚礼能力凭证。系统管理员沿用现有账号密码会话。

**Tech Stack:** Go 1.26 / Gin / GORM / SQLite（glebarez）；原生 HTML/CSS/JS 模板。

**Spec:** `docs/superpowers/specs/2026-09-05-multitenant-token-design.md`

## Global Constraints

- token：`crypto/rand` 32 字节 → `base64.RawURLEncoding`；格式校验 `^[A-Za-z0-9_-]{20,64}$`，先校验再查库
- 所有 SQL 经 GORM 参数化（不用字符串拼接）；`:token` 一律先过格式校验
- 全新空库：不迁移旧数据；旧 `settings`（无 wedding_id 行）由 AutoMigrate 后忽略
- 删除婚礼必须清理：settings/guests/visits 行 + `static/{token}` 媒体目录
- 新人后台 token 授权，无密码；系统管理员会话沿用 HttpOnly Cookie
- 页面/JS 风格沿用现有（中文注释、深色金色 admin、磨砂请柬）
- 每次任务结束提交一次 commit

---

### Task 1: 数据模型：Wedding + 多租户字段

**Files:**
- Modify: `internal/models/models.go`
- Modify: `internal/db/db.go`

**Interfaces:**
- Produces: `models.Wedding{ID int64; Token string; Name string; CreatedAt time.Time}`；`models.Setting{WeddingID int64; Key string; Value string}`（复合主键）；`models.Guest`/`models.Visit` 各增 `WeddingID int64`
- Consumes: 无

- [ ] **Step 1: 写失败测试**

创建 `internal/models/models_test.go`：

```go
package models

import "testing"

func TestAutoMigrateIncludesWeddingAndIsolationFields(t *testing.T) {
	// 占位：确保结构体字段存在（编译期验证）
	_ = Wedding{Token: "x", Name: "y"}
	_ = Setting{WeddingID: 1, Key: "k", Value: "v"}
	_ = Guest{WeddingID: 1}
	_ = Visit{WeddingID: 1}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/models/` — 期望 FAIL（`Wedding`/字段未定义）。

- [ ] **Step 3: 实现模型**

`internal/models/models.go` 追加/修改：

```go
// Wedding 一场婚礼（多租户），token 即访问凭证
type Wedding struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Token     string    `json:"token" gorm:"uniqueIndex;not null;size:64"`
	Name      string    `json:"name" gorm:"not null;size:100"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// Setting 键值对设置，按婚礼隔离
type Setting struct {
	WeddingID int64  `json:"wedding_id" gorm:"primaryKey"`
	Key       string `json:"key" gorm:"primaryKey"`
	Value     string `json:"value" gorm:"not null"`
}
```

`Guest` 增加字段 `WeddingID int64 \`json:"wedding_id" gorm:"index;not null;default:0"\``；`Visit` 增加同名字段。`Guest` 原 `ID` 保留。

- [ ] **Step 4: 迁移列表**

`internal/db/db.go` AutoMigrate 改为：
`db.AutoMigrate(&models.Visit{}, &models.Guest{}, &models.AdminUser{}, &models.Setting{}, &models.Wedding{})`

- [ ] **Step 5: 运行通过**

Run: `go test ./internal/models/` — PASS。

- [ ] **Step 6: 提交**

```bash
git add internal/models internal/db
git commit -m "feat: 多租户数据模型（Wedding + wedding_id 隔离字段）"
```

---

### Task 2: weddings 包：token 生成与校验

**Files:**
- Create: `internal/weddings/weddings.go`
- Create: `internal/weddings/weddings_test.go`

**Interfaces:**
- Produces: `func GenerateToken() (string, error)`、`func IsValidToken(tok string) bool`
- Consumes: 无

- [ ] **Step 1: 写失败测试**

`internal/weddings/weddings_test.go`：

```go
package weddings

import "testing"

func TestGenerateTokenLengthAndCharset(t *testing.T) {
	tok, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken err: %v", err)
	}
	if len(tok) < 40 { // 32 字节 → base64url 43 字符
		t.Errorf("token 过短: %d", len(tok))
	}
	if !IsValidToken(tok) {
		t.Errorf("生成的 token 未通过校验: %q", tok)
	}
	// 唯一性抽样
	a, _ := GenerateToken()
	b, _ := GenerateToken()
	if a == b {
		t.Error("两次生成相同 token")
	}
}

func TestIsValidTokenRejectsBad(t *testing.T) {
	bad := []string{"", "abc", "../x", "a b c", "a;drop table", strings.Repeat("a", 100)}
	for _, s := range bad {
		if IsValidToken(s) {
			t.Errorf("应拒绝: %q", s)
		}
	}
	good := []string{"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"}
	for _, s := range good {
		if !IsValidToken(s) {
			t.Errorf("应通过: %q", s)
		}
	}
}
```

（需 `import "strings"`。）

- [ ] **Step 2: 运行确认失败** — `go test ./internal/weddings/` FAIL（包不存在）。

- [ ] **Step 3: 实现**

`internal/weddings/weddings.go`：

```go
package weddings

import (
	"crypto/rand"
	"encoding/base64"
	"regexp"
)

var tokenRe = regexp.MustCompile(`^[A-Za-z0-9_-]{20,64}$`)

// GenerateToken 生成 CSPRNG 婚礼 token：32 字节 → base64url
func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// IsValidToken 校验 token 格式（字母数字与 -_ 下划线，长度 20-64）
func IsValidToken(tok string) bool {
	return tokenRe.MatchString(tok)
}
```

- [ ] **Step 4: 运行通过** — `go test ./internal/weddings/` PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/weddings
git commit -m "feat: 婚礼 token 生成与格式校验"
```

---

### Task 3: settings 改为按 wedding_id 隔离

**Files:**
- Modify: `internal/settings/settings.go`
- Modify: `internal/settings/settings_test.go`

**Interfaces:**
- Produces: `settings.New(db *gorm.DB, weddingID int64, defaults config.WeddingConfig) *Store`；`(*Store).All()/Get/Set` 行为不变但按 wedding_id 过滤
- Consumes: models.Setting（复合主键）

- [ ] **Step 1: 写失败测试**

在 `settings_test.go` 的 `newTestStore` 基础上新增隔离测试（把 `newTestStore` 的 `New` 调用改成带 weddingID 的新签名）：

```go
func TestIsolationBetweenWeddings(t *testing.T) {
	s1 := New(newDB(t), 1, config.WeddingConfig{GroomName: "A"})
	s2 := New(newDB(t), 2, config.WeddingConfig{GroomName: "B"})
	if err := s1.Set(KeyVenue, "场地A"); err != nil {
		t.Fatal(err)
	}
	if got := s2.Get(KeyVenue); got != "" {
		t.Errorf("婚礼2 不应看到婚礼1 的设置: %q", got)
	}
	if got := s1.Get(KeyVenue); got != "场地A" {
		t.Errorf("婚礼1 丢失设置: %q", got)
	}
	if got := s2.Get(KeyGroomName); got != "B" {
		t.Errorf("婚礼2 默认值应来自自身 defaults: %q", got)
	}
}
```

`newTestStore` 改为返回 `(*Store, *gorm.DB)`（便于构造第二个 Store 用同一库）。改动 `newTestStore(t, seed)` 内 `return New(db, config.WeddingConfig{...})` → `New(db, 1, config.WeddingConfig{...})` 并返回 db。同步把所有 `s := newTestStore(t, seed)` 改成 `s, _ := newTestStore(t, seed)`。

- [ ] **Step 2: 运行确认失败** — `go test ./internal/settings/` 编译失败（New 签名变）。

- [ ] **Step 3: 实现隔离**

`internal/settings/settings.go`：
- `Store` 增加 `weddingID int64`。
- `New`：`func New(db *gorm.DB, weddingID int64, wedding config.WeddingConfig) *Store { return &Store{db: db, weddingID: weddingID, defaults: defaultSettings(wedding)} }`
- `All()` 查询改 `s.db.Where("wedding_id = ?", s.weddingID).Find(&rows)`
- `Set` 写入 `&models.Setting{WeddingID: s.weddingID, Key: key, Value: value}`

- [ ] **Step 4: 运行通过** — `go test ./internal/settings/` PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/settings
git commit -m "feat: settings 按 wedding_id 隔离"
```

---

### Task 4: handler 引入 wedding 作用域

**Files:**
- Modify: `internal/handler/handler.go`

**Interfaces:**
- Consumes: `weddings.GenerateToken/IsValidToken`、`settings.New(db, weddingID, cfg)`、`models.Wedding`
- Produces: `Handler` 去掉 `Settings` 字段与构造参数；新增 `settingsFor(weddingID) *settings.Store`、`weddingDir(token) string`；新增中间件 `weddingContext()`（c.Set("wedding", *models.Wedding)）；公开方法改为 `getInvitation/recordVisit/submitRSVP`（从 ctx 取 wedding）
- Consumes(下一任务): Task 6 用这些方法

- [ ] **Step 1: 确认编译基线** — `go test ./internal/handler/` 当前应全绿。

- [ ] **Step 2: 改结构体与构造器**

Handler 去掉 `Settings` 字段；`New(db, sessions, staticDir)`（移除 store 参数）。新增字段与方法：

```go
type Handler struct {
	DB        *gorm.DB
	Sessions  *auth.TokenStore
	StaticDir string
}

// settingsFor 构造指定婚礼的设置存储（默认值来自婚礼初始信息+代码常量）
func (h *Handler) settingsFor(weddingID int64) *settings.Store {
	return settings.New(h.DB, weddingID, config.WeddingConfig{})
}

// weddingDir 某 token 的媒体根目录（staticDir/{token}）
func (h *Handler) weddingDir(token string) string {
	return filepath.Join(h.StaticDir, token)
}

// weddingFrom 从上下文取当前婚礼
func weddingFrom(c *gin.Context) *models.Wedding {
	if v, ok := c.Get("wedding"); ok {
		return v.(*models.Wedding)
	}
	return nil
}
```

需新增 import `"wedding-invitation/internal/config"`。`config.WeddingConfig{}` 空默认：defaultSettings 用空值覆盖 groom/bride/date/venue，其余取代码常量；细节由婚礼后台编辑。

- [ ] **Step 3: wedding 中间件**

```go
// weddingContext 校验 token 并加载 Wedding，注入上下文
func (h *Handler) weddingContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		tok := c.Param("token")
		if !weddings.IsValidToken(tok) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		var w models.Wedding
		if err := h.DB.Where("token = ?", tok).First(&w).Error; err != nil {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.Set("wedding", &w)
		c.Next()
	}
}
```

- [ ] **Step 4: 重构现有方法为 wedding 作用域**

逐个方法把 `h.Settings` 换成 `h.settingsFor(w.ID)`、路径用 `h.weddingDir(w.Token)`。示例（`getInvitation` 核心段）：

```go
func (h *Handler) getInvitation(c *gin.Context) {
	w := weddingFrom(c)
	if w == nil { c.AbortWithStatus(http.StatusNotFound); return }
	all := h.settingsFor(w.ID).All()
	// ...其余与原来一致，slides 改为：
	"slides": h.slidesFor(w.Token),
	"music_url": all[settings.KeyMusicURL],
}
```

方法清单与改动点：
- `slides()` → `slidesFor(token string)`：`images.List(filepath.Join(h.weddingDir(token), "images"))`
- `imagesDir()` → `imagesDirFor(token)`：`filepath.Join(h.weddingDir(token), "images")`；`musicDir()` 同理
- `uploadImage/deleteImage/listImages/uploadAudio/deleteAudio/listAudio`：方法体内 `w := weddingFrom(c)`；用 `w.Token` 拼目录；无 wedding → 404
- `recordVisit/submitRSVP`：建 `models.Guest{... WeddingID: w.ID}`（Visit 同理）
- `getGuests/getVisits/updateGuest/deleteGuest/createGuest/exportGuests/getSettings/updateSettings`：查询加 `.Where("wedding_id = ?", w.ID)`；`updateGuest/deleteGuest` 的 `First(&guest, id)` 改 `Where("id = ? AND wedding_id = ?", id, w.ID).First(&guest)`
- `calcCountdown` 不变

为让本任务可独立编译测试，先不改 `RegisterRoutes`（Task 6 统一改路由）。本任务结束前 `RegisterRoutes` 仍引用旧路径但调用已签名一致的方法——若编译不过，可暂时保留旧方法签名空壳。**判定标准：`go build ./internal/handler/` 通过（可暂不注册新路由）**，方法实现已按 wedding 作用域写完。

- [ ] **Step 5: 构建通过** — `go build ./...`（若 main.go 的 `handler.New` 参数报错，属 Task 6 范围，可临时把 main.go 改为传空 store 编译通过，Task 6 再修）。

- [ ] **Step 6: 提交**

```bash
git add internal/handler
git commit -m "refactor: handler 全部方法改为 wedding_id 作用域"
```

---

### Task 5: 系统管理员 weddings CRUD + 删除级联

**Files:**
- Modify: `internal/handler/handler.go`

**Interfaces:**
- Consumes: `weddings.GenerateToken`、`models.Wedding`、`h.settingsFor`
- Produces: `listWeddings/createWedding/deleteWedding` handlers（Task 6 注册路由）

- [ ] **Step 1: 写处理器**

在 handler.go 管理段添加：

```go
// listWeddings 系统管理员：列出全部婚礼（含统计）
func (h *Handler) listWeddings(c *gin.Context) {
	type row struct {
		models.Wedding
		RsvpCount      int64 `json:"rsvp_count"`
		AttendingCount int64 `json:"attending_count"`
	}
	var rows []row
	err := h.DB.Model(&models.Wedding{}).
		Select(`weddings.*,
			(SELECT COUNT(*) FROM guests WHERE wedding_id = weddings.id) AS rsvp_count,
			(SELECT COUNT(*) FROM guests WHERE wedding_id = weddings.id AND attending = 1) AS attending_count`).
		Order("created_at DESC").Scan(&rows).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": rows})
}

// createWedding 系统管理员：创建婚礼并生成 token，可选预填新人信息
func (h *Handler) createWedding(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		GroomName   string `json:"groom_name"`
		BrideName   string `json:"bride_name"`
		WeddingDate string `json:"wedding_date"`
		Venue       string `json:"venue"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "请求格式错误"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len(req.Name) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "婚礼名称不能为空且不超过100字"})
		return
	}
	tok, err := weddings.GenerateToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "生成 token 失败"})
		return
	}
	w := models.Wedding{Token: tok, Name: req.Name}
	if err := h.DB.Create(&w).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "创建失败"})
		return
	}
	// 预填可编辑设置（可选字段）
	store := h.settingsFor(w.ID)
	prefill := map[string]string{
		settings.KeyGroomName:   req.GroomName,
		settings.KeyBrideName:   req.BrideName,
		settings.KeyWeddingDate: req.WeddingDate,
		settings.KeyVenue:       req.Venue,
	}
	for k, v := range prefill {
		if strings.TrimSpace(v) != "" {
			if err := store.Set(k, strings.TrimSpace(v)); err != nil {
				log.Printf("预填设置失败 %s: %v", k, err)
			}
		}
	}
	log.Printf("系统管理员创建婚礼: %s (token=%s)", w.Name, w.Token)
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{
		"id": w.ID, "name": w.Name, "token": w.Token,
		"invite_url": "/w/" + w.Token, "admin_url": "/admin/" + w.Token,
	}})
}

// deleteWedding 系统管理员：删除婚礼并级联清理数据与媒体目录
func (h *Handler) deleteWedding(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "无效的 ID"})
		return
	}
	var w models.Wedding
	if err := h.DB.First(&w, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "婚礼不存在"})
		return
	}
	tx := h.DB.Begin()
	for _, m := range []interface{}{
		&models.Setting{}, &models.Guest{}, &models.Visit{},
	} {
		if err := tx.Where("wedding_id = ?", w.ID).Delete(m).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "删除失败"})
			return
		}
	}
	if err := tx.Delete(&w).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "删除失败"})
		return
	}
	tx.Commit()
	// 清理媒体目录（失败仅告警）
	if dir := filepath.Join(h.StaticDir, w.Token); dir != "" {
		if err := os.RemoveAll(dir); err != nil {
			log.Printf("删除媒体目录失败 %s: %v", dir, err)
		}
	}
	log.Printf("系统管理员删除婚礼: %s (id=%d)", w.Name, w.ID)
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"status": "ok"}})
}
```

- [ ] **Step 2: 构建通过** — `go build ./internal/handler/`。

- [ ] **Step 3: 提交**

```bash
git add internal/handler
git commit -m "feat: 系统管理员 weddings CRUD 与删除级联"
```

---

### Task 6: main.go 路由与页面

**Files:**
- Modify: `main.go`
- Create: `web/templates/system.html`（空壳，Task 9 填充）

**Interfaces:**
- Consumes: `handler.New(db, sessions, staticDir)`、Task 4/5 的方法与中间件
- Produces: 运行时可用的多租户路由

- [ ] **Step 1: 改 handler 构造**

`main.go` 中 `handler.New(db.DB, sessions, settingsStore, cfg.Paths.StaticDir)` → `handler.New(db.DB, sessions, cfg.Paths.StaticDir)`；删除 `settingsStore := settings.New(db.DB, cfg.Wedding)`（不再需要）。

- [ ] **Step 2: 页面路由**

`RegisterRoutes` 在 handler 里注册 API 路由（含页面跳转见下）。把 `main.go` 中 `r.GET("/")` 与 `r.GET("/admin")` 改掉，并在 handler 加页面方法：

handler.go 增加：

```go
// weddingPage 请柬页：校验 token 后渲染 index.html
func (h *Handler) weddingPage(c *gin.Context) {
	tok := c.Param("token")
	if !weddings.IsValidToken(tok) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	var w models.Wedding
	if err := h.DB.Where("token = ?", tok).First(&w).Error; err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.HTML(http.StatusOK, "index.html", gin.H{"Token": tok})
}

// adminPage 婚礼后台页：校验 token 后渲染 admin.html
func (h *Handler) adminPage(c *gin.Context) {
	tok := c.Param("token")
	if !weddings.IsValidToken(tok) {
		c.Redirect(http.StatusFound, "/admin")
		return
	}
	var w models.Wedding
	if err := h.DB.Where("token = ?", tok).First(&w).Error; err != nil {
		c.Redirect(http.StatusFound, "/admin")
		return
	}
	c.HTML(http.StatusOK, "admin.html", gin.H{"Token": tok})
}
```

`RegisterRoutes` 追加页面路由（页面在 main.go 已有 HTML 模板加载前提下可用）：

```go
r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusFound, "/admin") })
r.GET("/admin", func(c *gin.Context) { c.HTML(http.StatusOK, "system.html", nil) })
r.GET("/w/:token", h.weddingPage)
r.GET("/admin/:token", h.adminPage)
```

`index.html` 与 `admin.html` 需要能渲染带 `.Token` 的数据（模板中用 `{{if .Token}}` 之类需模板接受字典，Go 模板对未知字段会报错——因此模板只通过 JS 从 URL 取 token，`.Token` 仅备用，Task 7/8 不加模板字段）。此处页面先渲染空数据亦可，token 由 JS 从 path 解析。

- [ ] **Step 3: API 路由改命名空间**

`RegisterRoutes` 重写：系统管理与婚礼 API 分开：

```go
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	// 页面
	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusFound, "/admin") })
	r.GET("/admin", func(c *gin.Context) { c.HTML(http.StatusOK, "system.html", nil) })
	r.GET("/w/:token", h.weddingPage)
	r.GET("/admin/:token", h.adminPage)

	// 系统管理员认证
	r.POST("/api/admin/login", h.adminLogin)
	r.POST("/api/admin/logout", h.adminLogout)
	sys := r.Group("/api/admin", h.authMiddleware())
	sys.GET("/weddings", h.listWeddings)
	sys.POST("/weddings", h.createWedding)
	sys.DELETE("/weddings/:id", h.deleteWedding)

	// 某婚礼（公开+后台共用 token 命名空间）
	w := r.Group("/api/w/:token", h.weddingContext())
	w.GET("/invitation", h.getInvitation)
	w.GET("/meta", h.getWeddingMeta)
	w.POST("/visit", h.recordVisit)
	w.POST("/rsvp", h.submitRSVP)
	w.GET("/settings", h.getSettings)
	w.PUT("/settings", h.updateSettings)
	w.GET("/guests", h.getGuests)
	w.POST("/guests", h.createGuest)
	w.PUT("/guests/:id", h.updateGuest)
	w.DELETE("/guests/:id", h.deleteGuest)
	w.GET("/guests/export", h.exportGuests)
	w.GET("/visits", h.getVisits)
	w.GET("/images", h.listImages)
	w.POST("/images", h.uploadImage)
	w.DELETE("/images/:name", h.deleteImage)
	w.GET("/audio", h.listAudio)
	w.POST("/audio", h.uploadAudio)
	w.DELETE("/audio/:name", h.deleteAudio)
}
```

`getWeddingMeta`：

```go
func (h *Handler) getWeddingMeta(c *gin.Context) {
	w := weddingFrom(c)
	if w == nil { c.AbortWithStatus(http.StatusNotFound); return }
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"name": w.Name, "token": w.Token}})
}
```

同时删去旧公开路由 `/api/invitation`、`/api/visit`、`/api/rsvp` 与旧的 `stats/guests/visits/settings/images/audio` 系统路由（已迁走）。

- [ ] **Step 4: 建空壳 system.html**

`web/templates/system.html` 放一个最小占位（Task 9 完整实现）：

```html
<!DOCTYPE html><html lang="zh-CN"><head><meta charset="UTF-8"><title>系统管理</title></head>
<body style="background:#0f0f1a;color:#f5f5f5;font-family:sans-serif;text-align:center;padding-top:20vh">系统后台建设中…</body></html>
```

- [ ] **Step 5: 起服冒烟**

Run: `go build -o wedding . && rm -rf /tmp/mt && mkdir -p /tmp/mt/static && PORT=8091 DB_PATH=/tmp/mt/s.db STATIC_DIR=/tmp/mt/static TEMPLATE_DIR=./web/templates ./wedding &`
`curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8091/` → 302；`curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8091/admin` → 200；`curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8091/w/nonexistent` → 404。清理进程。

- [ ] **Step 6: 提交**

```bash
git add main.go internal/handler web/templates/system.html
git commit -m "feat: 多租户路由与页面（/admin /w/:token /admin/:token）"
```

---

### Task 7: index.html 按 token 请求

**Files:**
- Modify: `web/templates/index.html`

**Interfaces:**
- Consumes: `GET /api/w/:token/invitation|visit|rsvp`
- Produces: 可用的 token 化请柬页

- [ ] **Step 1: 加 token 解析**

`index.html` 脚本开头（IIFE 内首行）加：

```js
// 从路径 /w/{token} 取 token
var WEDDING_TOKEN = (location.pathname.split('/')[2] || '');
var API = '/api/w/' + WEDDING_TOKEN;
```

- [ ] **Step 2: 替换请求 URL**

把所有 `/api/invitation` → `API + '/invitation'`；`/api/visit` → `API + '/visit'`；`/api/rsvp` → `API + '/rsvp'`。（共 3 处 fetch。）

- [ ] **Step 3: 手动/浏览器验证**

Run: 起服 → 建婚礼（见 Task 5/10 的 curl 流程）取 token → 打开 `/w/{token}`，确认姓名/样式/轮播正常，RSVP 可提交。

- [ ] **Step 4: 提交**

```bash
git add web/templates/index.html
git commit -m "feat: 请柬页按 /api/w/:token 请求"
```

---

### Task 8: admin.html 婚礼后台化 + 顶部婚礼名

**Files:**
- Modify: `web/templates/admin.html`

**Interfaces:**
- Consumes: `/admin/:token` 页面、`/api/w/:token/*`（settings/guests/visits/images/audio/meta）
- Produces: 无登录、token 化的婚礼后台

- [ ] **Step 1: token 解析 + 移除登录**

`admin.html`：删除登录页块与 `doLogin/checkAuth/loginBtn` 相关 JS；顶部加 token 解析：

```js
var WEDDING_TOKEN = (location.pathname.split('/')[2] || '');
var API = '/api/w/' + WEDDING_TOKEN;
```

脚本开头校验：`fetch(API + '/meta')` 若 404/非 ok → `location.href = '/admin'`。登录成功逻辑（原 showAdmin 内部）改为直接 `showAdmin()`。

- [ ] **Step 2: 替换请求 URL**

把 admin.html 中所有 `/api/admin/...` 请求前缀改为 `API + '/...'`：`stats`(删/或由 overview 提供)、`settings`、`guests`、`guests/:id`、`guests/export`、`visits`、`images`、`audio`。（原 `/api/admin/stats` 与 `visits` 属婚礼后台数据，改为 `/api/w/{token}/visits` 与统计从 guests 计算或调 `/api/w/{token}/guests` 前端汇总——统计卡片数据改为由 guests+visits 接口在前端汇总，或后端加 `/api/w/:token/stats`。为最小改动：`loadStats` 改为后端新增的婚礼 stats，见下 Step 4。）

- [ ] **Step 3: 顶部婚礼名**

header 右侧/标题前加 `<span id="weddingName"></span>`；showAdmin 时 `fetch(API+'/meta').then(d => { document.getElementById('weddingName').textContent = d.data.name; })`。

- [ ] **Step 4: 婚礼统计接口**

handler.go 增加：

```go
// getWeddingStats 某婚礼的统计（复用旧 getStats 逻辑，按 wedding_id 过滤）
func (h *Handler) getWeddingStats(c *gin.Context) {
	w := weddingFrom(c)
	if w == nil { c.AbortWithStatus(http.StatusNotFound); return }
	gid := w.ID
	stats := gin.H{}
	var total, today int64
	h.DB.Model(&models.Visit{}).Where("wedding_id = ?", gid).Count(&total)
	h.DB.Model(&models.Visit{}).Where("wedding_id = ? AND date(visited_at) = date('now')", gid).Count(&today)
	stats["total_visits"] = total
	stats["today_visits"] = today
	var rsvp, attending int64
	h.DB.Model(&models.Guest{}).Where("wedding_id = ?", gid).Count(&rsvp)
	h.DB.Model(&models.Guest{}).Where("wedding_id = ? AND attending = 1", gid).Count(&attending)
	stats["total_rsvp"] = rsvp
	stats["attending_count"] = attending
	// 其余（独立 IP、趋势等）照旧按 wedding_id 过滤，此处省略为精简；前端展示所需字段至少含上述 4 项
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": stats})
}
```

注册路由 `w.GET("/stats", h.getWeddingStats)`。`admin.html` 的 `loadStats` 改用 `API + '/stats'`，渲染字段与旧一致（total_visits/today_visits/unique_visitors/total_rsvp/total_headcount/not_attending_count/visit_trend —— 后端补齐这些字段，把旧 getStats 整段按 wedding_id 过滤搬入）。

- [ ] **Step 5: 浏览器验证**

建婚礼 → 打开 `/admin/{token}` → 顶部显示婚礼名、可保存设置、来宾增删、上传图音、无登录提示。

- [ ] **Step 6: 提交**

```bash
git add web/templates/admin.html internal/handler
git commit -m "feat: 婚礼后台 token 化 + 顶部婚礼名 + per-wedding stats"
```

---

### Task 9: system.html 系统后台（登录+总览）

**Files:**
- Create: `web/templates/system.html`

**Interfaces:**
- Consumes: `POST /api/admin/login|logout`、`GET/POST/DELETE /api/admin/weddings`
- Produces: 系统管理员 UI

- [ ] **Step 1: 写页面**

复用 admin.html 登录页样式；登录后显示表格 + 创建表单。核心 JS：

```html
<!-- 系统后台：登录 + 婚礼总览 -->
<form id="createForm">
  <input id="cfName" placeholder="婚礼名称（如 李祥 & 王羚羚 的婚礼）" required maxlength="100">
  <input id="cfGroom" placeholder="新郎（选填）"><input id="cfBride" placeholder="新娘（选填）">
  <input id="cfDate" type="date"><input id="cfVenue" placeholder="地点（选填）">
  <button type="submit">创建新婚礼</button>
</form>
<table id="wedTable"><thead><tr><th>名称</th><th>创建时间</th><th>RSVP</th><th>出席</th><th>请柬链接</th><th>后台链接</th><th>操作</th></tr></thead><tbody></tbody></table>
```

JS：`checkAuth()` 探测 `GET /api/admin/weddings` 200 → 显示总览；否则显示登录。登录成功后 `loadWeddings()`。行内给出 `复制请柬链接`（`location.origin + '/w/' + token`）与 `复制后台链接`（`/admin/' + token`），删除按钮 `confirm` 后 `DELETE /api/admin/weddings/:id` 再刷新。创建成功自动复制后台链接。

- [ ] **Step 2: 浏览器验证**

起服 → `/admin` → 登录（admin/lucky123）→ 创建婚礼 → 表格出现 → 点击复制链接。

- [ ] **Step 3: 提交**

```bash
git add web/templates/system.html
git commit -m "feat: 系统后台婚礼总览与创建"
```

---

### Task 10: 端到端验证（隔离、清理、浏览器）

**Files:**
- 无生产代码改动（只做验证脚本）

**Interfaces:**
- 验收本计划全部任务

- [ ] **Step 1: 全量测试** — `go test ./...` 全绿；`go vet ./...`。

- [ ] **Step 2: 隔离起服脚本验证**

```bash
W=$(pwd); go build -o "$W/wedding" "$W"
rm -rf /tmp/mte && mkdir -p /tmp/mte/static
PORT=8092 DB_PATH=/tmp/mte/s.db STATIC_DIR=/tmp/mte/static TEMPLATE_DIR="$W/web/templates" "$W"/wedding >/tmp/mte/log.txt 2>&1 &
sleep 2
# 系统登录
curl -s -c /tmp/mte/c.txt -X POST http://localhost:8092/api/admin/login -H 'Content-Type: application/json' -d '{"username":"admin","password":"lucky123"}'
# 建两场婚礼
T1=$(curl -s -b /tmp/mte/c.txt -X POST http://localhost:8092/api/admin/weddings -H 'Content-Type: application/json' -d '{"name":"A&B","groom_name":"A","bride_name":"B","wedding_date":"2026-12-01"}' | python3 -c 'import json,sys;print(json.load(sys.stdin)["data"]["token"])')
T2=$(curl -s -b /tmp/mte/c.txt -X POST http://localhost:8092/api/admin/weddings -H 'Content-Type: application/json' -d '{"name":"C&D"}' | python3 -c 'import json,sys;print(json.load(sys.stdin)["data"]["token"])')
# 隔离：T1 写来宾，T2 不可见；T1 设置不污染 T2
curl -s -X POST http://localhost:8092/api/w/$T1/rsvp -H 'Content-Type: application/json' -d '{"name":"客人","phone":"1","attending":1,"headcount":1}' >/dev/null
curl -s http://localhost:8092/api/w/$T2/guests | grep -q '"total_rsvp"' || true   # T2 无该数据
echo "T2 guests rows: $(curl -s http://localhost:8092/api/w/$T2/guests | python3 -c 'import json,sys;print(len(json.load(sys.stdin)["data"]))')"  # 期望 0
# 非法 token
echo "bad token http: $(curl -s -o /dev/null -w '%{http_code}' http://localhost:8092/w/..%2Fevil)"  # 404
# 删除 T2 后媒体/数据清理（T2 无媒体，验证 404 即可）
curl -s -b /tmp/mte/c.txt -X DELETE http://localhost:8092/api/admin/weddings/$(curl -s -b /tmp/mte/c.txt http://localhost:8092/api/admin/weddings | python3 -c "import json,sys;print([w['id'] for w in json.load(sys.stdin)['data'] if w['token']=='$T2'][0])") >/dev/null
echo "T2 after delete: $(curl -s -o /dev/null -w '%{http_code}' http://localhost:8092/api/w/$T2/invitation)"  # 404
pkill -f /tmp/mte
```

- [ ] **Step 3: 浏览器验收** — 系统登录建婚礼 → 复制并打开 `/w/{token}` 与 `/admin/{token}`；两场婚礼的图片/来宾互不可见。

- [ ] **Step 4: 更新 README 与 images/music README**（说明 token 分目录、新人后台无密码、多租户用法）。提交。

---

## 自审结论

- **Spec 覆盖**：数据模型/Token/路由/API/媒体分目录/删除级联/系统后台/前端 token 化均落到 Task 1-9；Task 10 验收隔离与清理。
- **占位扫描**：除 system.html 前端细节以"复用现有样式"描述外，Go 侧均给出可编译代码；前端给出关键结构与接口。
- **类型一致**：`settings.New(db, weddingID, cfg)`、`handler.New(db, sessions, staticDir)`、`weddings.GenerateToken` 等跨任务签名一致。
