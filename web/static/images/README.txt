# 静态资源目录
# 此目录下的所有图片（jpg/jpeg/png/gif/webp/bmp）都会作为请柬页的背景轮播图，
# 按文件名排序展示，无需修改前端代码即可增删图片。
#
# 也可以通过管理后台（"设置" Tab）上传 / 删除背景图片。
#
# Docker 部署时，通过 volume 映射覆盖此目录：
#   -v /宿主机图片路径:/app/web/static/images
#
# 或在 docker-compose.yml 中配置：
#   volumes:
#     - ./my-photos:/app/web/static/images
