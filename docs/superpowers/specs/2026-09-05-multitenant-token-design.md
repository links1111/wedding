# 多租户 Token 隔离系统设计

日期：2026-09-05
状态：已与用户确认方向

## Context（背景）

现有系统是单租户婚礼请柬：一套全局设置（settings 键值）、一套来宾/访问记录、共享的 static/images 与 static/music 目录，管理后台用单一 admin 账号密码。无法让多对新人各自独立使用。

目标：改造为多租户——系统管理员为每对新人创建独立婚礼事件；每对新人通过唯一 Token 链接访问自己的请柬与管理后台，无需共享管理员密码；数据（请柬信息、RSVP、访问记录、图片、音乐）严格按婚礼隔离。

已确认决策：路由 `/w/{token}` + `/admin/{token}` + `/admin`（系统）；系统管理员（现有账号密码）+ Token 授权新人；全新空库（不迁移旧数据）；媒体按 token 分目录。

## 目标 / 非目标

目标：
- 每对新人一套完全独立的婚礼数据与媒体
- Token 即新人的凭证：链接打开请柬与后台，无密码
- 系统管理员在 `/admin` 总览创建/删除婚礼
- 尽可能复用现有 handler/模板/样式

非目标（本轮不做）：
- 每婚礼独立密码登录
- 多管理员角色/权限细分
- Token 过期轮换（管理员可删除重建代替）
- 用户注册/自助开通（由系统管理员代建）

## 数据模型

单 SQLite 库，按 `wedding_id` 隔离。

新增 `Wedding`：
```go
type Wedding struct {
    ID        int64     `gorm:"primaryKey;autoIncrement"`
    Token     string    `gorm:"uniqueIndex;not null;size:64"`
    Name      string    `gorm:"not null;size:100"`   // 展示名，如「李祥 & 王羚羚 的婚礼」
    CreatedAt time.Time `gorm:"autoCreateTime"`
}
```

改动：
- `Setting`：主键由 `Key` 改为复合主键 `(WeddingID int64, Key string)`；`Value` 不变。查询/写入全部带 `wedding_id`。
- `Guest`：新增 `WeddingID int64`，加索引；原 `IP` 等字段保留。
- `Visit`：新增 `WeddingID int64`，加索引。
- `AdminUser`：保留（系统管理员）。

数据库迁移：`AutoMigrate` 自动建新表/加列。因采用全新空库，无需迁移旧数据；如库中有旧 `settings`（无 wedding_id），启动时忽略或清空。

## Token

- 生成：`crypto/rand` 读 32 字节 → `base64.RawURLEncoding`（43 字符，字符集 `[A-Za-z0-9_-]`）。
- 校验：任意 `:token` 路径参数先过 `^[A-Za-z0-9_-]{20,64}$`，不匹配直接 404/400（防路径遍历与无谓查询），再查库取 Wedding。
- 凭证语义：token 即该婚礼的完整能力——可访问请柬公开数据，也可访问该婚礼后台（settings/guests/images 等）。链接持有者可改后台。属本模型固有权衡，建议仅 HTTPS 传播；系统管理员可删除婚礼重建以作废 token。

## 路由与页面

| 路径 | 页面 | 说明 |
|------|------|------|
| `/` | — | 301 重定向到 `/admin` |
| `/admin` | 系统后台（新 `system.html`） | 登录 + 婚礼总览表格 +「创建新婚礼」+ 每行删除 |
| `/w/:token` | 请柬（`index.html`） | 公开访问，URL 带 token |
| `/admin/:token` | 婚礼后台（`admin.html`） | 顶部显示当前婚礼名；Token 授权 |

前端统一从 `location.pathname.split('/')[2]` 取 token（`/w/x` 与 `/admin/x` 均为第 2 段）。

模板加载沿用现有：优先 `TEMPLATE_DIR`，回退 embed.FS。

## API

### 系统管理员（AdminUser 会话，现有 auth）
- `POST /api/admin/login`、`POST /api/admin/logout`（沿用）
- `GET /api/admin/weddings` —— 列表：每场含 name、token、创建时间、RSVP 数/出席数（聚合）
- `POST /api/admin/weddings` —— 创建：body 含 `name`；生成 token；按配置默认值种子化该婚礼 settings
- `DELETE /api/admin/weddings/:id` —— 删除：级联删除 settings/guests/visits + 删除媒体目录 `static/{token}/…`

旧的单婚礼 `/api/admin/stats|guests|visits` 从系统后台移除（其能力迁入 per-wedding API）。

### 该婚礼公开
- `GET /api/w/:token/invitation` —— 原 `getInvitation`，返回该婚礼 settings 合并的文案/样式、`slides`（该 token 的 images 目录）、`music_url`、`map_link`
- `POST /api/w/:token/visit` —— 记录访问（带 wedding_id）
- `POST /api/w/:token/rsvp` —— 提交回执（带 wedding_id）

### 该婚礼管理（token 授权，与公开同命名空间）
- `GET|PUT /api/w/:token/settings`
- `GET /api/w/:token/guests`、`POST|PUT|DELETE /api/w/:token/guests/:id`、`GET /api/w/:token/guests/export`
- `GET /api/w/:token/visits`
- `GET|POST /api/w/:token/images`、`DELETE /api/w/:token/images/:name`
- `GET|POST /api/w/:token/audio`、`DELETE /api/w/:token/audio/:name`

所有查询按 `wedding_id` 过滤（GORM 参数化）。上传/删除的目录 = `staticDir/{token}/images|music`（沿用 images/audio 包的安全校验，含扩展名白名单、防路径穿越、去重、大小限制）。

### 作用域解析
新增中间件 `weddingContext`：对 `/w/:token` 与 `/api/w/:token` 校验 token 格式并加载 Wedding，注入 context；404 处理不存在的 token。

## 媒体按 token 分目录

- 图片：`web/static/{token}/images`
- 音乐：`web/static/{token}/music`
- 轮播 slides 与 music_url 默认 `/static/{token}/images/…`、`/static/{token}/music/bgm.mp3`
- `r.Static("/static", staticDir)` 继续服务全部子目录
- settings 里 `music_url` 等若以 `/static/` 开头，视为相对根路径；前端拼 URL 时无需改（本来就相对站点根）

## 前端改动

- `index.html`：JS 读取 token → 请求改为 `/api/w/{token}/…`；其余样式/手势/音乐逻辑不变。
- `admin.html`：读取 token → 请求 `/api/w/{token}/…`；顶部 header 增加当前婚礼名（由 invitation/婚礼信息接口或注入给出）；登录 UI 移除（改为 Token 授权），若 token 无效跳 `/admin`。
- 新增 `system.html`：系统管理员登录 + 婚礼总览表格（名称/创建时间/统计）、「创建新婚礼」表单、删除按钮（confirm）。
- 后台媒体上传控件：上传路径按当前 token。

## 错误处理与安全

- token 格式白名单（先校验再查库）
- 参数化查询（GORM）——不拼接 SQL
- 删除婚礼：事务删除关联行 + `os.RemoveAll` 媒体目录；失败回滚/记录
- 保留现有安全响应头、Cookie Secure（系统管理员会话走 HTTPS）
- 上传限制、路径穿越防护沿用 images/audio 包
- 系统管理员会话沿用 HttpOnly Cookie

## 测试

- 单测：token 生成长度/字符集、token 格式校验；settings 按 wedding_id 隔离（两个婚礼的键互不可见）；Wedding CRUD
- 集成（隔离库/静态目录起服 + curl）：创建婚礼 → 拿到 token → 该婚礼公开/管理 API 正常；两个 token 的数据互不可见；删除婚礼后数据与媒体目录清理；非法 token 404
- 浏览器：系统登录创建婚礼 → `/w/:token` 请柬正常 → `/admin/:token` 后台可配置 → 两个 token 隔离

## 涉及文件（预估）

- `internal/models/models.go`（Wedding + 字段）
- `internal/settings`（按 wedding_id）
- `internal/handler`（全部方法按 wedding 作用域；新增 weddings CRUD、token 中间件）
- `internal/auth`（沿用，系统管理员）
- `internal/config`（可能加 token 长度等常量）
- `main.go`（路由、/ 重定向、system.html）
- `web/templates/index.html`、`admin.html`（token 化 + header 婚礼名）
- `web/templates/system.html`（新）
- `web/static/images/README.txt`、`web/static/music/README.txt`（说明按 token 分目录）
- README 更新

## 采用方法说明

之所以选"单库 + wedding_id 隔离"而非"每婚礼独立库/实例"：系统管理员需要在 `/admin` 一个总览里管理所有婚礼；单库配合隔离查询可满足，且复用现有 SQLite/GORM 结构与前端。token 即凭证而非"每婚礼密码"，符合"新人用链接直接进后台、免密码"的目标。
