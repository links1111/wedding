# 💍 婚礼电子请柬

> 一个基于 Go + SQLite 的轻量级婚礼电子请柬系统，单二进制部署，零外部依赖。

包含精美的请柬展示页面和功能完整的管理后台。前后端不分离，HTML 模板通过 `embed.FS` 嵌入二进制文件，编译后只需一个可执行文件即可运行。

## ✨ 功能特性

### 请柬页面（访客端）
- 🎨 深色金色主题，优雅的衬线字体设计
- ✉️ 开场信封动画，点击打开请柬
- 🕐 实时婚礼倒计时
- 📝 RSVP 在线回复（出席/缺席、人数、祝福留言）
- 🌟 Canvas 粒子背景动画
- 📱 移动端优先响应式设计

### 管理后台（管理员端）
- 📊 统计仪表盘：总访问量、今日访问、独立访客、RSVP 统计、出席人数
- 📈 近 7 天访问趋势柱状图
- 👥 来宾列表管理，支持搜索筛选
- 📋 访问记录查看
- 📥 来宾数据 CSV 导出（Excel 兼容）
- 🔐 bcrypt 密码加密 + HttpOnly Cookie 会话认证

### 安全特性
- bcrypt 密码哈希存储
- CSPRNG 随机会话 Token
- HttpOnly + SameSite Cookie
- 安全响应头（X-Content-Type-Options、X-Frame-Options、Referrer-Policy）
- 输入长度限制与校验
- 请求超时保护

## 🛠 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.23+ · 标准库 `net/http` |
| 数据库 | SQLite（`go-sqlite3` 驱动，WAL 模式） |
| 认证 | bcrypt 密码哈希 + 内存会话管理 |
| 前端 | 原生 HTML/CSS/JavaScript（无框架依赖） |
| 部署 | 单二进制文件（`embed.FS` 嵌入模板） |

## 📦 项目结构

```
wedding/
├── main.go                      # 程序入口，路由注册，优雅关闭
├── go.mod                       # Go 模块定义
├── internal/
│   ├── config/
│   │   ├── config.go            # 环境变量配置加载
│   │   └── random.go            # CSPRNG 随机 token 生成
│   ├── db/
│   │   └── db.go                # SQLite 初始化与建表
│   ├── models/
│   │   └── models.go            # 数据模型定义
│   ├── auth/
│   │   ├── auth.go              # 管理员认证与会话管理
│   │   └── token.go             # 会话 token 生成
│   └── handler/
│       └── handler.go           # HTTP 处理器与 API 路由
└── web/
    └── templates/
        ├── index.html           # 请柬页面
        └── admin.html           # 管理后台页面
```

## 🚀 快速开始

### 环境要求

- **Go** 1.23+
- **GCC**（`go-sqlite3` 需要 CGO 编译）
  - Windows: 安装 [MinGW-w64](https://www.mingw-w64.org/) 或 [TDM-GCC](https://jmeubank.github.io/tdm-gcc/)
  - Linux: `apt install gcc` 或 `yum install gcc`
  - macOS: `xcode-select --install`

### 编译运行

```bash
# 设置 Go 代理（国内环境推荐）
export GOPROXY=https://goproxy.cn,direct

# 下载依赖
go mod tidy

# 编译
go build -o wedding-invitation .

# 运行（使用默认配置）
./wedding-invitation
```

> 💡 Windows 环境下使用 `go build -o wedding-invitation.exe .` 编译，运行 `.\wedding-invitation.exe`。

首次启动时，如果未通过环境变量设置管理员密码，系统会自动生成随机密码并打印到控制台。

### 自定义配置

通过环境变量配置新人信息和婚礼详情：

```bash
export GROOM_NAME="张三"
export BRIDE_NAME="李四"
export WEDDING_DATE="2025-10-01"
export WEDDING_VENUE="北京·国贸大酒店"
export ADMIN_USER="admin"
export ADMIN_PASS="your-secure-password"
export JWT_SECRET="your-jwt-secret"
export PORT="8080"
export DB_PATH="./data/wedding.db"

./wedding-invitation
```

> 🔐 生产环境务必显式设置 `ADMIN_PASS` 和 `JWT_SECRET`，不要依赖自动生成的随机值。

### 访问页面

| 页面 | 地址 |
|------|------|
| 请柬页面 | `http://localhost:8080/` |
| 管理后台 | `http://localhost:8080/admin` |

## 📡 API 接口

### 公开接口

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/invitation` | 获取请柬信息（新人、日期、场地、倒计时） |
| `POST` | `/api/visit` | 记录访客访问 |
| `POST` | `/api/rsvp` | 提交 RSVP 回复 |

**RSVP 请求示例：**

```json
{
  "name": "王五",
  "phone": "13800138000",
  "attending": 1,
  "headcount": 2,
  "message": "祝你们新婚快乐！"
}
```

> `attending`: `1` = 出席，`2` = 缺席

### 管理接口（需登录）

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/admin/login` | 管理员登录 |
| `POST` | `/api/admin/logout` | 退出登录 |
| `GET` | `/api/admin/stats` | 统计数据 |
| `GET` | `/api/admin/guests` | 来宾列表 |
| `GET` | `/api/admin/visits` | 访问记录 |
| `GET` | `/api/admin/guests/export` | 导出来宾 CSV |

## 🗄 数据库表结构

### `visits` — 访问记录

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | INTEGER PK | 自增主键 |
| `ip` | TEXT | 访客 IP |
| `user_agent` | TEXT | 浏览器标识 |
| `referer` | TEXT | 来源页面 |
| `visitor_name` | TEXT | 访客姓名 |
| `visited_at` | TEXT | 访问时间 |
| `created_at` | TEXT | 记录创建时间 |

### `guests` — RSVP 回复

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | INTEGER PK | 自增主键 |
| `name` | TEXT | 来宾姓名 |
| `phone` | TEXT | 联系电话 |
| `attending` | INTEGER | 出席状态：0=未确认 1=出席 2=缺席 |
| `headcount` | INTEGER | 出席人数 |
| `message` | TEXT | 祝福留言 |
| `ip` | TEXT | 提交 IP |
| `created_at` | TEXT | 创建时间 |
| `updated_at` | TEXT | 更新时间 |

### `admin_users` — 管理员

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | INTEGER PK | 自增主键 |
| `username` | TEXT UNIQUE | 用户名 |
| `password_hash` | TEXT | bcrypt 密码哈希 |
| `created_at` | TEXT | 创建时间 |

## 🌐 部署

### 直接部署

```bash
# 交叉编译（Linux 服务器）
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o wedding-invitation .

# 上传到服务器后运行
./wedding-invitation
```

> ⚠️ `go-sqlite3` 依赖 CGO，编译时需要安装 GCC（Linux: `apt install gcc`，macOS: `xcode-select --install`，Windows: 安装 MinGW-w64）。

### 使用 Docker 部署

```dockerfile
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY . .
RUN CGO_ENABLED=1 go build -o wedding-invitation .

FROM alpine:latest
RUN apk add --no-cache libc6-compat
WORKDIR /app
COPY --from=builder /app/wedding-invitation .
EXPOSE 8080
VOLUME ["/app/data"]
CMD ["./wedding-invitation"]
```

```bash
docker build -t wedding-invitation .
docker run -d -p 8080:8080 \
  -e GROOM_NAME="张三" \
  -e BRIDE_NAME="李四" \
  -e WEDDING_DATE="2025-10-01" \
  -e WEDDING_VENUE="北京·国贸大酒店" \
  -e ADMIN_PASS="your-secure-password" \
  -v wedding-data:/app/data \
  wedding-invitation
```

### 使用 Nginx 反向代理（推荐）

```nginx
server {
    listen 80;
    server_name wedding.example.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 使用 systemd 管理服务

```ini
[Unit]
Description=Wedding Invitation
After=network.target

[Service]
Type=simple
User=www
WorkingDirectory=/opt/wedding
ExecStart=/opt/wedding/wedding-invitation
Environment=GROOM_NAME=张三
Environment=BRIDE_NAME=李四
Environment=WEDDING_DATE=2025-10-01
Environment=WEDDING_VENUE=北京·国贸大酒店
Environment=ADMIN_PASS=your-secure-password
Environment=JWT_SECRET=your-jwt-secret
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## ⚙️ 环境变量参考

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `PORT` | `8080` | 服务监听端口 |
| `DB_PATH` | `./data/wedding.db` | SQLite 数据库路径 |
| `ADMIN_USER` | `admin` | 管理员用户名 |
| `ADMIN_PASS` | 随机生成 | 管理员密码（建议显式设置） |
| `JWT_SECRET` | 随机生成 | 会话签名密钥（建议显式设置） |
| `GROOM_NAME` | `新郎` | 新郎姓名 |
| `BRIDE_NAME` | `新娘` | 新娘姓名 |
| `WEDDING_DATE` | `2025-10-01` | 婚礼日期（YYYY-MM-DD） |
| `WEDDING_VENUE` | `婚礼殿堂` | 婚礼场地 |

## 📝 License

MIT

## 🔒 安全说明

- **密码安全**：管理员密码使用 bcrypt 哈希存储，不明文保存。首次启动若未设置 `ADMIN_PASS`，系统自动生成随机密码并打印到控制台，请及时记录并修改。
- **会话安全**：管理后台使用 CSPRNG 生成的随机 Token 进行会话管理，通过 HttpOnly + SameSite Cookie 传递，定期自动清理过期会话。
- **输入校验**：所有外部输入（RSVP 表单、访客记录）在服务端进行类型、长度、范围校验，数据库操作全部使用参数化查询，防止 SQL 注入。
- **安全响应头**：全局添加 `X-Content-Type-Options: nosniff`、`X-Frame-Options: DENY`、`Referrer-Policy` 等安全头，防止点击劫持和 MIME 嗅探。
- **请求超时**：服务器配置读写超时（15s）和空闲超时（60s），防止慢速攻击。
- **密钥管理**：`JWT_SECRET` 和 `ADMIN_PASS` 建议通过环境变量显式设置，不要使用自动生成的随机值用于生产环境。

## ❓ FAQ

<details>
<summary>编译报错 <code>cgo: C compiler "gcc" not found</code></summary>

`go-sqlite3` 依赖 CGO 编译，需要安装 GCC：

- **Windows**：安装 [MinGW-w64](https://www.mingw-w64.org/) 或 [TDM-GCC](https://jmeubank.github.io/tdm-gcc/)
- **Linux**：`apt install gcc` 或 `yum install gcc`
- **macOS**：`xcode-select --install`

</details>

<details>
<summary>如何修改已部署的管理员密码？</summary>

删除 `data/wedding.db` 文件并重启服务，系统会重新初始化数据库并创建管理员账户。或直接设置 `ADMIN_PASS` 环境变量后重启。

</details>

<details>
<summary>数据存储在哪里？</summary>

所有数据存储在 `DB_PATH` 指定的 SQLite 文件中（默认 `./data/wedding.db`）。SQLite 使用 WAL 模式，同目录下会有 `wedding.db-wal` 和 `wedding.db-shm` 文件。备份时需同时复制这三个文件。

</details>

<details>
<summary>可以不用 CGO 吗？</summary>

可以。将 `go-sqlite3` 替换为纯 Go 实现的 `modernc.org/sqlite` 驱动，无需 GCC 即可编译。但需要修改 `internal/db/db.go` 中的驱动导入路径。

</details>

## 🤝 贡献

欢迎提交 Issue 或 Pull Request。

## 📸 截图预览

> 请柬页面：深色金色主题，信封开场动画，粒子背景，实时倒计时
>
> 管理后台：统计仪表盘，7 天趋势图，来宾列表，CSV 导出

---

<p align="center">Made with 💕 · 祝天下有情人终成眷属</p>
