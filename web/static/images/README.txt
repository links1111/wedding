# 背景图片目录（多租户）
# 每个婚礼一个子目录：web/static/images/{婚礼请柬token}/，例如：
#   web/static/images/abcdef.../slide1.jpg
# 该婚礼的轮播图会展示其 token 目录下的所有图片（jpg/jpeg/png/gif/webp/bmp）。
# 正常使用请通过管理后台「复制请柬链接」对应的婚礼后台 → 设置 → 背景图片 上传，会自动落到对应目录。
#
# Docker 部署时通过 volume 映射整个静态目录：
#   -v /宿主机目录:/app/web/static
