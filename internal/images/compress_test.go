package images

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

// makeJPEG 构造一个指定尺寸的 JPEG 测试图
func makeJPEG(t *testing.T, w, h, quality int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 128, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		t.Fatalf("编码测试图片失败: %v", err)
	}
	return buf.Bytes()
}

func TestCompressToJPEGResizes(t *testing.T) {
	in := makeJPEG(t, 4000, 3000, 95)
	out, err := CompressToJPEG(bytes.NewReader(in), 1920, 82)
	if err != nil {
		t.Fatalf("CompressToJPEG 报错: %v", err)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("输出无法解码: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("输出格式 = %s, 期望 jpeg", format)
	}
	if cfg.Width != 1920 || cfg.Height != 1440 {
		t.Errorf("输出尺寸 = %dx%d, 期望 1920x1440", cfg.Width, cfg.Height)
	}
	if len(out) >= len(in) {
		t.Errorf("压缩后未变小: in=%d out=%d", len(in), len(out))
	}
}

func TestCompressToJPEGNoUpscale(t *testing.T) {
	in := makeJPEG(t, 800, 600, 90)
	out, err := CompressToJPEG(bytes.NewReader(in), 1920, 82)
	if err != nil {
		t.Fatalf("CompressToJPEG 报错: %v", err)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("输出无法解码: %v", err)
	}
	if cfg.Width != 800 || cfg.Height != 600 {
		t.Errorf("小图不应被放大: %dx%d", cfg.Width, cfg.Height)
	}
}

func TestCompressToJPEGPngInput(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	out, err := CompressToJPEG(bytes.NewReader(buf.Bytes()), 1920, 82)
	if err != nil {
		t.Fatalf("CompressToJPEG PNG 输入报错: %v", err)
	}
	if _, format, _ := image.DecodeConfig(bytes.NewReader(out)); format != "jpeg" {
		t.Errorf("PNG 输入输出格式 = %s, 期望 jpeg", format)
	}
}

func TestCompressToJPEGTransparentBecomesWhite(t *testing.T) {
	// 半透明像素压缩后不应是黑色（白底合成）
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	img.SetRGBA(25, 25, color.RGBA{255, 0, 0, 128})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	out, err := CompressToJPEG(bytes.NewReader(buf.Bytes()), 1920, 82)
	if err != nil {
		t.Fatalf("CompressToJPEG 报错: %v", err)
	}
	decoded, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("输出无法解码: %v", err)
	}
	// 取角落像素（透明区域），应接近白色而非黑色
	r, g, b, _ := decoded.At(5, 5).RGBA()
	if r < 60000 || g < 60000 || b < 60000 {
		t.Errorf("透明区域应接近白色, 实际 (%d,%d,%d)", r>>8, g>>8, b>>8)
	}
}

func TestCompressToJPEGInvalidInput(t *testing.T) {
	if _, err := CompressToJPEG(bytes.NewReader([]byte("not an image")), 1920, 82); err == nil {
		t.Error("非法输入应报错")
	}
}

func TestCompressedName(t *testing.T) {
	cases := map[string]string{
		"photo.PNG":  "photo.jpg",
		"bg.webp":    "bg.jpg",
		"a.bmp":      "a.jpg",
		"slide1.JPG": "slide1.jpg",
		"noext":      "noext.jpg",
		"x.jpeg":     "x.jpg",
	}
	for in, want := range cases {
		if got := CompressedName(in); got != want {
			t.Errorf("CompressedName(%q) = %q, 期望 %q", in, got, want)
		}
	}
}
