# 静态资源目录
# 将婚礼背景图片放置在此目录下，文件名对应 index.html 中的引用：
#   images/slide1.jpg
#   images/slide2.jpg
#   images/slide3.jpg
#   images/slide4.jpg
#
# Docker 部署时，通过 volume 映射覆盖此目录：
#   -v /宿主机图片路径:/app/web/static/images
#
# 或在 docker-compose.yml 中配置：
#   volumes:
#     - ./my-photos:/app/web/static/images
