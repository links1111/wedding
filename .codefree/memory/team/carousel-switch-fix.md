---
name: carousel-switch-fix
type: project
scope: team
description: 婚礼请柬系统轮播图切换修复方案： 核心问题：在 performFadeSwitch 切换卡片时，旧图片尚未移除，carShow 函数通过 querySelectorAll('.slide') 获取的幻灯片数量与 carList 长度不一致，导致 carIdx 计算错乱，定时器启动后图片无法正确切换。...
created: "2026-09-08T03:34:53.284Z"
updated: "2026-09-08T03:45:50.882Z"
---
婚礼请柬系统轮播图切换修复方案： 核心问题：在 performFadeSwitch 切换卡片时，旧图片尚未移除，carShow 函数通过 querySelectorAll('.slide') 获取的幻灯片数量与 carList 长度不一致，导致 carIdx 计算错乱，定时器启动后图片无法正确切换。 修复方案： 1. 在切换前预加载新卡片的图片列表，确保后续轮播流畅。 2. 在过渡完成后（0.8 秒），移除旧图片，给新图片的第一张添加 active 类并设置 opacity 为 1。 3. 关键修复：设置所有新图片的 opacity 为 1，而不仅仅是第一张，确保其他图片在轮播时可见。 4. 更新 carList 和 carIdx，重新启动定时器。 5. 平滑交叉淡入淡出：滚动到新卡片时，旧图片淡出的同时新图片第一张淡入，不等待预加载完成，过渡时间 0.8 秒。 6. 自动播放：切换后轮播图需自动继续播放，定时器在淡入淡出完成后重新启动。