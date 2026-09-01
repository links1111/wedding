package images

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestIsAllowedImage(t *testing.T) {
	allowed := []string{"a.jpg", "a.JPG", "a.jpeg", "a.png", "a.gif", "a.webp", "a.bmp"}
	for _, name := range allowed {
		if !IsAllowedImage(name) {
			t.Errorf("IsAllowedImage(%q) 应返回 true", name)
		}
	}
	notAllowed := []string{"a.txt", "a", "a.mp4", ".jpg"}
	for _, name := range notAllowed {
		if IsAllowedImage(name) {
			t.Errorf("IsAllowedImage(%q) 应返回 false", name)
		}
	}
}

func TestSanitizeName(t *testing.T) {
	// 路径中的目录部分应被剥除
	if got, err := SanitizeName("../images/evil.png"); err != nil || got != "evil.png" {
		t.Errorf("SanitizeName(../images/evil.png) = %q, %v", got, err)
	}
	if got, err := SanitizeName("C:\\tmp\\pic.JPG"); err != nil || got != "pic.JPG" {
		t.Errorf("SanitizeName(C:\\\\tmp\\\\pic.JPG) = %q, %v", got, err)
	}
	if _, err := SanitizeName("notes.txt"); err == nil {
		t.Error("SanitizeName 非图片扩展名应报错")
	}
	if _, err := SanitizeName(""); err == nil {
		t.Error("SanitizeName 空名应报错")
	}
	if _, err := SanitizeName("  .."); err == nil {
		t.Error("SanitizeName 纯目录应报错")
	}
}

func TestSafeJoin(t *testing.T) {
	dir := t.TempDir()
	// 正常文件
	p, err := SafeJoin(dir, "slide1.JPG")
	if err != nil {
		t.Fatalf("SafeJoin 正常文件报错: %v", err)
	}
	if want := filepath.Join(dir, "slide1.JPG"); p != want {
		t.Errorf("SafeJoin = %q, 期望 %q", p, want)
	}
	// 路径穿越应报错
	for _, evil := range []string{"../../etc/passwd", "..", "../secret", "a/../../b"} {
		if _, err := SafeJoin(dir, evil); err == nil {
			t.Errorf("SafeJoin(%q) 应报错", evil)
		}
	}
	// 绝对路径注入应被拒绝
	if _, err := SafeJoin(dir, filepath.Join("/tmp", "x.png")); err == nil {
		t.Errorf("SafeJoin(%q) 绝对路径应报错", filepath.Join("/tmp", "x.png"))
	}
	// 嵌套路径也应被拒绝
	if _, err := SafeJoin(dir, "sub/x.png"); err == nil {
		t.Error("SafeJoin 嵌套路径应报错")
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"slide2.JPG", "slide1.JPG", "note.txt", "bg.png", "subdir"} {
		if name == "subdir" {
			if err := os.Mkdir(filepath.Join(dir, name), 0755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := List(dir)
	if err != nil {
		t.Fatalf("List 报错: %v", err)
	}
	want := []string{"bg.png", "slide1.JPG", "slide2.JPG"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("List = %v, 期望 %v", got, want)
	}
}

func TestListMissingDir(t *testing.T) {
	if _, err := List(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("List 不存在的目录应报错")
	}
}
