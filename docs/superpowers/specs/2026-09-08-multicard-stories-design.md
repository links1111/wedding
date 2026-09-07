# 多卡片"我们的故事"设计

日期：2026-09-08
状态：已与用户确认

## Context

婚礼请柬目前只有主请柬一张卡。希望展示多段"我们的故事"（第一次相遇、旅行、求婚、订婚…），每段是一张独立的子卡片，各有自己的背景图与文案；后台可增删改、排序、启停；子卡片样式与背景音乐**复用主请柬**（不单独设置）。浏览方式改为**上下滚动整页**；原"上下滑调模糊"手势改为**长按逐渐模糊、松手恢复**。

## 目标 / 非目标

目标：
- 每婚礼一组子卡片（主请柬仍由 settings 驱动，不属于 cards 表）
- 预设类型（第一次相遇/旅行/求婚/订婚/自定义）+ 每张内容可编辑（标题/日期/正文）
- 每张子卡自选背景图（来自该婚礼图片库），与主卡**复用同一套卡片样式与背景音乐**
- 请柬页上下滚动浏览主请柬→各子卡→末尾回执
- 手势：长按任意处 → 模糊随按住时长渐增，松手恢复
- 后台：设置页顶部 tab 列表展示各卡（含主请柬），可添加（选类型/自定义）、点卡进独立编辑页（内容+背景图+启用+删除），可排序

非目标：子卡独立样式/独立音乐设置；多选/批量；图片再上传（复用现有背景图库）。

## 数据模型

新增表 `cards`（每婚礼）：
```go
type Card struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	WeddingID int64  `gorm:"index;not null"`
	Sort      int    `gorm:"not null;default:0"`
	Type      string `gorm:"size:32;not null;default:'custom'"` // preset key 或 custom
	Title     string `gorm:"size:100;not null;default:''"`
	Date      string `gorm:"size:50;not null;default:''"`
	Content   string `gorm:"size:2000;not null;default:''"`
	Images    string `gorm:"size:2000;not null;default:''"` // 本卡背景图文件名，逗号分隔（从该婚礼 images 目录取）
	Enabled   bool   `gorm:"not null;default:true"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}
```
预设 key：`first_meet / travel / proposal / engagement / custom`，UI 提供中文选择，添加时预填标题与文案占位（travel 预填"旅行"标题，可多次添加）。

AutoMigrate 加入 `models.Card{}`。

## API

### 后台（/api/a/:token）
- `GET /api/a/:token/cards` —— 该婚礼全部卡片（按 sort）
- `POST /api/a/:token/cards` —— 新增（body: type/title/date/content/images/enabled）；Sort 取当前 max+1
- `PUT /api/a/:token/cards/:id` —— 更新字段；`POST /api/a/:token/cards/reorder` body {ids:[...]} 按序重排 sort
- `DELETE /api/a/:token/cards/:id` —— 删除
- 校验：title≤100、date≤50、content≤2000、images 为白名单图片文件名列表（校验扩展名与存在于该婚礼 images 目录，避免任意路径）

### 公开（/api/w/:token）
- `/api/w/:token/invitation` 响应新增 `cards: [{id,type,title,date,content,images:[url,...]}]`，仅 enabled、按 sort
- images URL = `/static/{token}/images/{name}`

## 请柬页（index.html）

结构改为上下滚动的全屏卡片序列：
- 容器改可滚动（放开 overflow），保留背景；每屏一个 section：
  - 主请柬（复用现有内容；回执按钮改为滚动到末尾回执卡）
  - 每个启用子卡：自身背景轮播（若干图交叉淡化）+ 磨砂内容块（复用主卡样式变量与文字自适应/亮度渐变逻辑），展示 date/title/content
  - 末尾回执卡：完整回执表单 + 提交
- 每张子卡独立背景：给每 section 一个背景层，各自轮播其 images
- 手势改长按：`pointerdown` 起计时器，按住期间 `--glass-blur` 自配置值渐增到 +30px；`pointerup/leave` 取消并平滑恢复（复用现有 applyStyle/restore 思路）；移除原"上下滑调模糊/淡出"逻辑

适配：移动端为主；不破坏自动播放、亮度自适应、导航、音乐等既有功能。

## 后台（admin.html）

设置视图顶部：横向 tab 列表 = `主请柬 | 各子卡标题… | ＋添加`
- 点某卡：下方切换为该卡编辑区
- 主请柬编辑区 = 现有设置内容（不加样式/音乐/背景的卡片逻辑）
- 子卡编辑区字段：类型（下拉，新建时选）、标题、日期、正文、背景图片（该婚礼图片库勾选多张）、是否启用、保存/删除、上移下移
- 「＋添加」：弹窗/内联选类型（预设或自定义）→ 新建并入编辑

## 错误处理与安全

- 参数化（GORM）；长度校验；images 文件名经 images.IsAllowedImage 校验且在婚礼目录内
- 删除/排序按 wedding_id 作用域

## 测试

- 单测：cards 校验（标题长度、images 白名单/存在性）
- 集成：后台建多张卡 → 公开 invitation.cards 仅返回 enabled 且图 URL 正确；删除/排序生效；隔离（另一婚礼不可见）
- 浏览器：后台建"第一次相遇/旅行/求婚" → 请柬滚动查看、各卡背景图不同、长按模糊、末尾回执提交

## 涉及文件（预估）

- `internal/models/models.go`（Card）
- `internal/handler/handler.go`（cards CRUD + invitation 附 cards + reorder + 校验）
- `internal/images`（复用：IsAllowedImage/List）
- `web/templates/admin.html`（卡片 tab + 编辑）
- `web/templates/index.html`（滚动化 + 每卡背景 + 长按模糊 + 回执末尾）
