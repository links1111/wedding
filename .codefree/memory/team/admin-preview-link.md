---
name: admin-preview-link
type: project
scope: team
description: "婚礼请柬系统管理后台查看请柬跳转逻辑：管理后台通过 /admin/{admin_token} 访问，admin_token 不是请柬 token。；必须从 /api/a/:token/meta 接口返回数据中的 token 字段获取真正的请柬 token。"
created: "2026-09-08T01:19:52.409Z"
updated: "2026-09-08T02:50:36.678Z"
---
婚礼请柬系统管理后台查看请柬跳转逻辑：
1. 管理后台通过 /admin/{admin_token} 访问，admin_token 不是请柬 token。
2. 必须从 /api/a/:token/meta 接口返回数据中的 token 字段获取真正的请柬 token。
3. 构建跳转链接时使用 /w/{wedding_token} 格式，其中 wedding_token 为接口返回的 token。
4. 管理页面右上角的"查看请柬"按钮应与总览页面的预览按钮逻辑一致，确保跳转正确。
5. 前端代码需正确解析 URL 中的 admin_token，调用 API 获取 wedding_token，再构建跳转链接。