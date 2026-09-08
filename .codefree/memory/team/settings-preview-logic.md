---
name: settings-preview-logic
type: project
scope: team
description: 婚礼请柬系统设置页面预览功能逻辑： - 主请柬编辑时：左侧预览栏显示主请柬完整预览（调用 renderMainCardPreview），右侧显示主请柬配置表单。 - 子卡片编辑时：左侧预览栏显示对应子卡片预览（调用 renderCardPreview），右侧显示子卡片编辑表单。 - 预览栏显示/隐藏...
created: "2026-09-08T02:38:40.351Z"
updated: "2026-09-08T02:38:40.351Z"
---
婚礼请柬系统设置页面预览功能逻辑： - 主请柬编辑时：左侧预览栏显示主请柬完整预览（调用 renderMainCardPreview），右侧显示主请柬配置表单。 - 子卡片编辑时：左侧预览栏显示对应子卡片预览（调用 renderCardPreview），右侧显示子卡片编辑表单。 - 预览栏显示/隐藏控制：switchMainCard() 显示预览栏并渲染主请柬预览；openCard() 显示预览栏并渲染子卡片预览。 - HTML 结构要求：.settings-content 包含 .preview-sidebar 和 #cardPane；#cardPane 内部嵌套 .settings-form-area，无多余闭合标签。