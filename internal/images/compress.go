package images

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"image/jpeg"
	"io"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
)

// CompressedName 将文件名扩展名替换为 .jpg（压缩输出的目标名）
func CompressedName(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name)) + ".jpg"
}

// CompressToJPEG 读取图片，长边超过 maxDim 时按比例缩小，统一编码为 JPEG。
// 透明区域以白色为底合成，避免转 JPEG 后变黑。
func CompressToJPEG(r io.Reader, maxDim, quality int) ([]byte, error) {
	src, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("无法解码图片: %w", err)
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("图片尺寸无效")
	}

	// 等比缩放，保证长边不超过 maxDim
	if maxDim > 0 && (w > maxDim || h > maxDim) {
		if w >= h {
			h = int(float64(h) * float64(maxDim) / float64(w))
			w = maxDim
		} else {
			w = int(float64(w) * float64(maxDim) / float64(h))
			h = maxDim
		}
		if w <= 0 || h <= 0 {
			return nil, fmt.Errorf("缩放后尺寸无效")
		}
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	// 先铺白底，再用高质量缩放合成，保留视觉细节
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("编码 JPEG 失败: %w", err)
	}
	return buf.Bytes(), nil
}
