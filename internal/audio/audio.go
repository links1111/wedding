package audio

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// allowedExts 允许作为背景音乐的音频扩展名
var allowedExts = map[string]bool{
	".mp3":  true,
	".wav":  true,
	".ogg":  true,
	".m4a":  true,
	".aac":  true,
	".flac": true,
}

// IsAllowedAudio 判断文件扩展名是否为允许的音频类型（大小写不敏感）
func IsAllowedAudio(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	if !allowedExts[ext] {
		return false
	}
	return strings.TrimSuffix(name, filepath.Ext(name)) != ""
}

// SanitizeName 从上传文件名中提取安全 basename 并校验扩展名
func SanitizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "\\", "/")
	base := filepath.Base(name)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "", fmt.Errorf("无效的文件名")
	}
	if !IsAllowedAudio(base) {
		return "", fmt.Errorf("不支持的音频格式")
	}
	return base, nil
}

// SafeJoin 拼接目录与文件名，要求结果为 dir 的直接子项（防路径穿越）
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

// List 列出目录下的所有音频文件名（排序，忽略子目录与非音频文件）
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
		if IsAllowedAudio(e.Name()) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}
