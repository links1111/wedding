package handler

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"wedding-invitation/internal/audio"
	"wedding-invitation/internal/auth"
	"wedding-invitation/internal/images"
	"wedding-invitation/internal/models"
	"wedding-invitation/internal/settings"
)

// Handler 聚合所有 HTTP 处理器依赖
type Handler struct {
	DB        *gorm.DB
	Sessions  *auth.TokenStore
	Settings  *settings.Store
	StaticDir string
}

// New 创建 Handler
func New(db *gorm.DB, sessions *auth.TokenStore, store *settings.Store, staticDir string) *Handler {
	return &Handler{
		DB:        db,
		Sessions:  sessions,
		Settings:  store,
		StaticDir: staticDir,
	}
}

// maxUploadBytes 背景图片上传大小上限（压缩场景常见手机原图较大）
const maxUploadBytes = 20 << 20 // 20MB

// 压缩参数：长边上限与 JPEG 质量
const (
	maxImageDim  = 1920
	jpegQuality  = 82
)

// RegisterRoutes 在 Gin 引擎上注册所有路由
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	// 公开 API
	r.GET("/api/invitation", h.getInvitation)
	r.POST("/api/visit", h.recordVisit)
	r.POST("/api/rsvp", h.submitRSVP)

	// 管理 API
	r.POST("/api/admin/login", h.adminLogin)
	r.POST("/api/admin/logout", h.adminLogout)
	adminAuth := r.Group("/api/admin", h.authMiddleware())
	adminAuth.GET("/stats", h.getStats)
	adminAuth.GET("/guests", h.getGuests)
	adminAuth.GET("/visits", h.getVisits)
	adminAuth.GET("/guests/export", h.exportGuests)
	adminAuth.GET("/settings", h.getSettings)
	adminAuth.PUT("/settings", h.updateSettings)
	adminAuth.GET("/images", h.listImages)
	adminAuth.POST("/images", h.uploadImage)
	adminAuth.DELETE("/images/:name", h.deleteImage)
	adminAuth.GET("/audio", h.listAudio)
	adminAuth.POST("/audio", h.uploadAudio)
	adminAuth.DELETE("/audio/:name", h.deleteAudio)
}

// --- 公开 API ---

// getInvitation 返回请柬信息（文案、样式、背景图列表）
func (h *Handler) getInvitation(c *gin.Context) {
	all := h.Settings.All()
	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"data": gin.H{
			"groom":         all[settings.KeyGroomName],
			"bride":         all[settings.KeyBrideName],
			"date":          all[settings.KeyWeddingDate],
			"date_main":     all[settings.KeyDateMain],
			"date_sub":      all[settings.KeyDateSub],
			"venue":         all[settings.KeyVenue],
			"chinese_title": all[settings.KeyChineseTitle],
			"info_items": []gin.H{
				{"title": all[settings.KeyCeremonyTitle], "sub": all[settings.KeyCeremonySub], "time": all[settings.KeyCeremonyTime]},
				{"title": all[settings.KeyDinnerTitle], "sub": all[settings.KeyDinnerSub], "time": all[settings.KeyDinnerTime]},
				{"title": all[settings.KeyDressTitle], "sub": all[settings.KeyDressSub], "time": ""},
			},
			"footer":    []string{all[settings.KeyFooter1], all[settings.KeyFooter2], all[settings.KeyFooter3]},
			"handwrite": all[settings.KeyHandwrite],
			"countdown": h.calcCountdown(all[settings.KeyWeddingDate]),
			"slides":    h.slides(),
			"map_link":  all[settings.KeyMapLink],
			"music_url": all[settings.KeyMusicURL],
			"style": gin.H{
				"card_transparency": all[settings.KeyCardTransparency],
				"glass_enabled":     all[settings.KeyGlassEnabled],
				"glass_blur":        all[settings.KeyGlassBlur],
				"glass_saturate":    all[settings.KeyGlassSaturate],
				"card_color":        all[settings.KeyCardColor],
			},
		},
	})
}

// slides 列出静态目录 images 下所有图片的访问 URL
func (h *Handler) slides() []string {
	names, err := images.List(filepath.Join(h.StaticDir, "images"))
	if err != nil {
		return nil
	}
	urls := make([]string, 0, len(names))
	for _, n := range names {
		urls = append(urls, "/static/images/"+n)
	}
	return urls
}

// recordVisit 记录访问
func (h *Handler) recordVisit(c *gin.Context) {
	var req struct {
		VisitorName string `json:"visitor_name"`
	}
	// 即使解析失败也记录访问
	_ = c.ShouldBindJSON(&req)

	// 限制 visitor_name 长度
	if len(req.VisitorName) > 100 {
		req.VisitorName = req.VisitorName[:100]
	}

	ip := clientIP(c)
	ua := c.Request.Header.Get("User-Agent")
	if len(ua) > 500 {
		ua = ua[:500]
	}
	referer := c.Request.Header.Get("Referer")
	if len(referer) > 500 {
		referer = referer[:500]
	}

	if err := h.DB.Create(&models.Visit{
		IP:          ip,
		UserAgent:   ua,
		Referer:     referer,
		VisitorName: req.VisitorName,
	}).Error; err != nil {
		log.Printf("记录访问失败: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"status": "ok"}})
}

// submitRSVP 提交 RSVP
func (h *Handler) submitRSVP(c *gin.Context) {
	var req struct {
		Name      string `json:"name"`
		Phone     string `json:"phone"`
		Attending int    `json:"attending"`
		Headcount int    `json:"headcount"`
		Message   string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "请求格式错误"})
		return
	}

	// 输入校验
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len(req.Name) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "姓名不能为空且不超过50字"})
		return
	}
	if len(req.Phone) > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "电话号码过长"})
		return
	}
	if req.Attending < 0 || req.Attending > 2 {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "出席状态无效"})
		return
	}
	if req.Headcount < 1 || req.Headcount > 20 {
		req.Headcount = 1
	}
	if len(req.Message) > 500 {
		req.Message = req.Message[:500]
	}

	ip := clientIP(c)

	if err := h.DB.Create(&models.Guest{
		Name:      req.Name,
		Phone:     req.Phone,
		Attending: req.Attending,
		Headcount: req.Headcount,
		Message:   req.Message,
		IP:        ip,
	}).Error; err != nil {
		log.Printf("保存 RSVP 失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "保存失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"status": "ok"}})
}

// --- 管理 API ---

// adminLogin 管理员登录
func (h *Handler) adminLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "请求格式错误"})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || len(req.Username) > 50 || len(req.Password) > 200 {
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": auth.ErrInvalidCredentials.Error()})
		return
	}

	user, err := auth.VerifyAdmin(h.DB, req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": err.Error()})
		return
	}

	token := auth.GenerateToken()
	if token == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "生成会话失败"})
		return
	}
	h.Sessions.Create(token, 24*time.Hour)

	// 设置 HttpOnly cookie
	c.SetCookie("admin_session", token, 86400, "/", "", c.Request.TLS != nil, true)

	log.Printf("管理员登录成功: %s (ID: %d)", user.Username, user.ID)
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"username": user.Username}})
}

// adminLogout 管理员登出
func (h *Handler) adminLogout(c *gin.Context) {
	cookie, err := c.Cookie("admin_session")
	if err == nil && cookie != "" {
		h.Sessions.Destroy(cookie)
	}

	c.SetCookie("admin_session", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"status": "ok"}})
}

// getStats 获取统计数据
func (h *Handler) getStats(c *gin.Context) {
	stats := gin.H{}

	// 总访问数
	var totalVisits int64
	h.DB.Model(&models.Visit{}).Count(&totalVisits)
	stats["total_visits"] = totalVisits

	// 今日访问
	var todayVisits int64
	h.DB.Model(&models.Visit{}).Where("date(visited_at) = date('now')").Count(&todayVisits)
	stats["today_visits"] = todayVisits

	// 独立 IP 数
	var uniqueIPs int64
	h.DB.Model(&models.Visit{}).Distinct("ip").Count(&uniqueIPs)
	stats["unique_visitors"] = uniqueIPs

	// 总 RSVP 数
	var totalRSVP int64
	h.DB.Model(&models.Guest{}).Count(&totalRSVP)
	stats["total_rsvp"] = totalRSVP

	// 出席人数
	var attendingCount int64
	h.DB.Model(&models.Guest{}).Where("attending = 1").Count(&attendingCount)
	stats["attending_count"] = attendingCount

	// 出席总人数
	var totalHeadcount int64
	h.DB.Model(&models.Guest{}).Where("attending = 1").Select("COALESCE(SUM(headcount), 0)").Scan(&totalHeadcount)
	stats["total_headcount"] = totalHeadcount

	// 缺席人数
	var notAttending int64
	h.DB.Model(&models.Guest{}).Where("attending = 2").Count(&notAttending)
	stats["not_attending_count"] = notAttending

	// 最近7天访问趋势
	type trendItem struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	}
	var trend []trendItem
	h.DB.Model(&models.Visit{}).
		Select("date(visited_at) as date, COUNT(*) as count").
		Where("visited_at >= datetime('now', '-7 days')").
		Group("date(visited_at)").
		Order("date(visited_at)").
		Scan(&trend)

	trendResult := make([]gin.H, 0, len(trend))
	for _, item := range trend {
		trendResult = append(trendResult, gin.H{"date": item.Date, "count": item.Count})
	}
	stats["visit_trend"] = trendResult

	c.JSON(http.StatusOK, gin.H{"ok": true, "data": stats})
}

// getGuests 获取来宾列表
func (h *Handler) getGuests(c *gin.Context) {
	var guests []models.Guest
	if err := h.DB.Order("created_at DESC").Limit(500).Find(&guests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "data": guests})
}

// getVisits 获取访问记录
func (h *Handler) getVisits(c *gin.Context) {
	var visits []models.Visit
	if err := h.DB.Order("visited_at DESC").Limit(500).Find(&visits).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "data": visits})
}

// exportGuests 导出来宾 CSV
func (h *Handler) exportGuests(c *gin.Context) {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=guests.csv")
	// BOM for Excel
	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})
	c.Writer.Write([]byte("姓名,电话,出席状态,人数,留言,提交时间\n"))

	rows, err := h.DB.Model(&models.Guest{}).
		Select("name, phone, attending, headcount, message, created_at").
		Order("created_at DESC").
		Rows()
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var name, phone, message, createdAt string
		var attending, headcount int
		rows.Scan(&name, &phone, &attending, &headcount, &message, &createdAt)
		label := models.AttendingLabel(attending)
		// CSV 转义：字段中的逗号和引号
		line := csvEscape(name) + "," + csvEscape(phone) + "," + csvEscape(label) + "," +
			itoa(headcount) + "," + csvEscape(message) + "," + csvEscape(createdAt) + "\n"
		c.Writer.Write([]byte(line))
	}
}

// --- 设置管理 ---

// getSettings 返回全部可配置设置（默认值 + DB 覆盖）
func (h *Handler) getSettings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": h.Settings.All()})
}

// updateSettings 批量更新设置（先全量校验，再写入）
func (h *Handler) updateSettings(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "请求格式错误"})
		return
	}
	if len(req) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "没有可更新的设置"})
		return
	}
	for key, val := range req {
		if err := settings.Validate(key, val); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
			return
		}
	}
	for key, val := range req {
		if err := h.Settings.Set(key, val); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "保存设置失败"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": h.Settings.All()})
}

// --- 背景图片管理 ---

// imagesDir 静态图片目录
func (h *Handler) imagesDir() string {
	return filepath.Join(h.StaticDir, "images")
}

// listImages 列出背景图片
func (h *Handler) listImages(c *gin.Context) {
	names, err := images.List(h.imagesDir())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "读取图片目录失败"})
		return
	}
	items := make([]gin.H, 0, len(names))
	for _, n := range names {
		items = append(items, gin.H{"name": n, "url": "/static/images/" + n})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": items})
}

// uploadImage 上传背景图片到 static/images；compress=true 时压缩到最佳大小后保存
func (h *Handler) uploadImage(c *gin.Context) {
	// 限制请求体大小，防止超大文件
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "请选择要上传的图片"})
		return
	}
	defer file.Close()

	name, err := images.SanitizeName(header.Filename)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
		return
	}
	if header.Size > maxUploadBytes {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "图片大小不能超过 20MB"})
		return
	}

	// 先读入内存：压缩需要，且便于压缩失败时回退保存原图
	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "读取图片失败"})
		return
	}

	compress := c.PostForm("compress") == "true"
	if compress {
		name = images.CompressedName(name)
		if compressed, cerr := images.CompressToJPEG(bytes.NewReader(data), maxImageDim, jpegQuality); cerr == nil {
			data = compressed
		} else {
			log.Printf("压缩图片失败，保存原图 %s: %v", header.Filename, cerr)
		}
	}

	dir := h.imagesDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "创建图片目录失败"})
		return
	}

	// 同名文件已存在时自动追加序号，避免覆盖已有图片
	dst := name
	for i := 2; ; i++ {
		p, jerr := images.SafeJoin(dir, dst)
		if jerr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": jerr.Error()})
			return
		}
		if _, serr := os.Stat(p); os.IsNotExist(serr) {
			break
		}
		ext := filepath.Ext(name)
		dst = strings.TrimSuffix(name, ext) + "_" + strconv.Itoa(i) + ext
	}

	dstPath, err := images.SafeJoin(dir, dst)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
		return
	}
	out, err := os.Create(dstPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "保存图片失败"})
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, bytes.NewReader(data)); err != nil {
		_ = os.Remove(dstPath)
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "保存图片失败"})
		return
	}

	log.Printf("管理员上传背景图片: %s (compress=%v)", dst, compress)
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"name": dst, "url": "/static/images/" + dst}})
}

// deleteImage 删除背景图片
func (h *Handler) deleteImage(c *gin.Context) {
	name := c.Param("name")
	// 仅允许删除图片文件，且必须为 images 目录的直接子项
	if !images.IsAllowedImage(name) {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "非法的文件名"})
		return
	}
	p, err := images.SafeJoin(h.imagesDir(), name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "非法的文件名"})
		return
	}
	if err := os.Remove(p); err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "文件不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "删除失败"})
		return
	}
	log.Printf("管理员删除背景图片: %s", name)
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"status": "ok"}})
}

// --- 背景音乐管理 ---

// musicDir 背景音乐目录
func (h *Handler) musicDir() string {
	return filepath.Join(h.StaticDir, "music")
}

// listAudio 列出背景音乐文件
func (h *Handler) listAudio(c *gin.Context) {
	names, err := audio.List(h.musicDir())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "读取音乐目录失败"})
		return
	}
	items := make([]gin.H, 0, len(names))
	for _, n := range names {
		items = append(items, gin.H{"name": n, "url": "/static/music/" + n})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": items})
}

// uploadAudio 上传背景音乐到 static/music
func (h *Handler) uploadAudio(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "请选择要上传的音频"})
		return
	}
	defer file.Close()

	name, err := audio.SanitizeName(header.Filename)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
		return
	}
	if header.Size > maxUploadBytes {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "音频大小不能超过 20MB"})
		return
	}

	dir := h.musicDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "创建音乐目录失败"})
		return
	}

	// 同名文件已存在时自动追加序号
	dst := name
	for i := 2; ; i++ {
		p, jerr := audio.SafeJoin(dir, dst)
		if jerr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": jerr.Error()})
			return
		}
		if _, serr := os.Stat(p); os.IsNotExist(serr) {
			break
		}
		ext := filepath.Ext(name)
		dst = strings.TrimSuffix(name, ext) + "_" + strconv.Itoa(i) + ext
	}

	dstPath, err := audio.SafeJoin(dir, dst)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
		return
	}
	out, err := os.Create(dstPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "保存音频失败"})
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		_ = os.Remove(dstPath)
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "保存音频失败"})
		return
	}

	log.Printf("管理员上传背景音乐: %s", dst)
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"name": dst, "url": "/static/music/" + dst}})
}

// deleteAudio 删除背景音乐
func (h *Handler) deleteAudio(c *gin.Context) {
	name := c.Param("name")
	if !audio.IsAllowedAudio(name) {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "非法的文件名"})
		return
	}
	p, err := audio.SafeJoin(h.musicDir(), name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "非法的文件名"})
		return
	}
	if err := os.Remove(p); err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "文件不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "删除失败"})
		return
	}
	log.Printf("管理员删除背景音乐: %s", name)
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": gin.H{"status": "ok"}})
}

// --- 中间件 ---

// authMiddleware 管理员认证中间件
func (h *Handler) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie("admin_session")
		if err != nil || cookie == "" || !h.Sessions.Valid(cookie) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "未授权"})
			return
		}
		c.Next()
	}
}

// --- 工具函数 ---

func (h *Handler) calcCountdown(date string) int {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return 0
	}
	days := int(t.Sub(time.Now()).Hours() / 24)
	if days < 0 {
		days = 0
	}
	return days
}

func clientIP(c *gin.Context) string {
	// 优先从 X-Forwarded-For 获取（反向代理场景）
	if xff := c.Request.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	if xri := c.Request.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	// 去掉端口
	addr := c.Request.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx > 0 {
		addr = addr[:idx]
	}
	return addr
}

func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
