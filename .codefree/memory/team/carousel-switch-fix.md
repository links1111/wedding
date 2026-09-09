---
name: carousel-switch-fix
type: project
scope: team
description: "婚礼请柬系统轮播图切换修复方案： 根本原因：performFadeSwitch 过渡完成后，所有新图片的 inline style opacity 被设为 '1'，覆盖了 CSS .slide { opacity: 0 } 和 .slide.active { opacity: 1 } 的样式控制，导..."
created: "2026-09-08T03:34:53.284Z"
updated: "2026-09-08T04:01:12.647Z"
---
婚礼请柬系统轮播图切换修复方案： 根本原因：performFadeSwitch 过渡完成后，所有新图片的 inline style opacity 被设为 '1'，覆盖了 CSS .slide { opacity: 0 } 和 .slide.active { opacity: 1 } 的样式控制，导致 active 类切换失效，所有图片叠在一起。 修复方案：在 0.8 秒过渡结束后，清除所有新图片的 inline style opacity 和 transition，让 CSS class 重新控制可见性。 轮播图自动播放间隔：4 秒。

核心问题：在 performFadeSwitch 切换卡片时，旧图片尚未移除，carShow 函数通过 querySelectorAll('.slide') 获取的幻灯片数量与 carList 长度不一致，导致 carIdx 计算错乱，定时器启动后图片无法正确切换。

修复方案：
1. 在切换前预加载新卡片的图片列表，确保后续轮播流畅。
2. 在过渡完成后（0.8 秒），移除旧图片，给新图片的第一张添加 active 类并设置 opacity 为 1。
3. 关键修复：设置所有新图片的 opacity 为 1，而不仅仅是第一张，确保其他图片在轮播时可见。
4. 更新 carList 和 carIdx，重新启动定时器。
5. 平滑交叉淡入淡出：滚动到新卡片时，旧图片淡出的同时新图片第一张淡入，不等待预加载完成，过渡时间 0.8 秒。
6. 自动播放：切换后轮播图需自动继续播放，定时器在淡入淡出完成后重新启动。

二次修复（inline style 覆盖 CSS class 问题）：
- 根因：performFadeSwitch 的 setTimeout 中将所有新图片的 inline style opacity 设为 '1'，由于 inline style 优先级高于 CSS class，导致 .slide 的 CSS opacity:0 被覆盖，所有图片同时可见叠在一起，active 类切换完全失效，看起来图片不轮播。
- 修复：过渡完成后清除所有新图片的 inline style.opacity 和 style.transition，让 CSS class 重新控制可见性（.slide opacity:0，.slide.active opacity:1）。
- carShow 函数中 carIdx 直接使用传入参数 i，不再重新计算（carNext 传入的 i 已是正确索引）。

轮播间隔：5 秒（setInterval 5000ms）。