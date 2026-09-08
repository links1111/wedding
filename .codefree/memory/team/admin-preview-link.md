---
name: admin-preview-link
type: project
scope: team
description: "婚礼请柬系统管理后台查看请柬跳转逻辑：管理后台通过 /admin/{admin_token} 访问，admin_token 不是请柬 token。必须从 /api/a/:token/meta 接口返回数据中的 token 字段获取真正的请柬 token，并使用该 token 构建跳转链接 /w/{W..."
created: "2026-09-08T01:19:52.409Z"
updated: "2026-09-08T03:08:25.813Z"
---
婚礼请柬系统管理后台查看请柬跳转逻辑：管理后台通过 /admin/{admin_token} 访问，admin_token 不是请柬 token。必须从 /api/a/:token/meta 接口返回数据中的 token 字段获取真正的请柬 token，并使用该 token 构建跳转链接 /w/{WEDDING_TOKEN}。跳转逻辑需与总览页面的预览按钮保持一致，使用标准 <a> 标签 href 属性，不阻止默认行为。