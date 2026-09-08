---
name: carousel-switch-fix
type: project
scope: team
description: 婚礼请柬系统轮播图切换修复方案： 核心问题：在 performFadeSwitch 切换卡片时，旧图片尚未移除，carShow 函数通过 querySelectorAll('.slide') 获取的幻灯片数量与 carList 长度不一致，导致 carIdx 计算错乱，定时器启动后图片无法正确切换。...
created: "2026-09-08T03:34:53.284Z"
updated: "2026-09-08T03:34:53.284Z"
---
婚礼请柬系统轮播图切换修复方案： 核心问题：在 performFadeSwitch 切换卡片时，旧图片尚未移除，carShow 函数通过 querySelectorAll('.slide') 获取的幻灯片数量与 carList 长度不一致，导致 carIdx 计算错乱，定时器启动后图片无法正确切换。 修复方案：修改 carShow 函数，不再依赖 DOM 元素数量计算索引，而是直接使用全局 carList.length 作为索引计算的基准，确保 carIdx 始终在有效范围内，保证定时器触发时能正确切换当前卡片的图片。