package images

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// allowedExts 允许作为背景图片的扩展名
var allowedExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
	".bmp":  true,
}

// IsAllowedImage 判断文件名扩展名是否为允许的图片类型（大小写不敏感），
// 且文件名具有非空主名（排除 ".jpg" 这类纯扩展名文件）
func IsAllowedImage(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	if !allowedExts[ext] {
		return false
	}
	return strings.TrimSuffix(name, filepath.Ext(name)) != ""
}

// SanitizeName 从上传文件名中提取安全的 basename（兼容 / 与 \ 路径），并校验扩展名
func SanitizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	// 同时识别 / 和 \ 为路径分隔符，避免 Windows 风格路径绕过
	name = strings.ReplaceAll(name, "\\", "/")
	base := filepath.Base(name)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "", fmt.Errorf("无效的文件名")
	}
	if !IsAllowedImage(base) {
		return "", fmt.Errorf("不支持的图片格式")
	}
	return base, nil
}

// SafeJoin 拼接目录与文件名，要求结果为 dir 的直接子项
// （拒绝路径穿越、嵌套路径与绝对路径注入）
func SafeJoin(dir, name string) (string, error) {
	clean := filepath.Clean(filepath.Join(dir, name))
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	absClean, err := filepath.Abs(clean)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absDir, absClean)
	if err != nil {
		return "", err
	}
	if rel == "." || rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) ||
		strings.Contains(rel, string(filepath.Separator)) {
		return "", fmt.Errorf("非法的路径")
	}
	return clean, nil
}

// List 列出目录下的所有图片文件名（排序，忽略子目录与非图片文件）
func List(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if IsAllowedImage(e.Name()) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}
