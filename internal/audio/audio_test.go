package audio

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestIsAllowedAudio(t *testing.T) {
	allowed := []string{"bgm.mp3", "song.MP3", "a.wav", "a.ogg", "a.m4a", "a.aac", "a.flac"}
	for _, n := range allowed {
		if !IsAllowedAudio(n) {
			t.Errorf("IsAllowedAudio(%q) 应为 true", n)
		}
	}
	notAllowed := []string{"a.txt", "a.jpg", "a", ".mp3", "a.mp4"}
	for _, n := range notAllowed {
		if IsAllowedAudio(n) {
			t.Errorf("IsAllowedAudio(%q) 应为 false", n)
		}
	}
}

func TestSanitizeName(t *testing.T) {
	if got, err := SanitizeName("../dir/bgm.mp3"); err != nil || got != "bgm.mp3" {
		t.Errorf("SanitizeName(../dir/bgm.mp3) = %q, %v", got, err)
	}
	if got, err := SanitizeName("C:\\tmp\\song.MP3"); err != nil || got != "song.MP3" {
		t.Errorf("SanitizeName(win路径) = %q, %v", got, err)
	}
	if _, err := SanitizeName("a.jpg"); err == nil {
		t.Error("非音频扩展名应报错")
	}
	if _, err := SanitizeName(""); err == nil {
		t.Error("空名应报错")
	}
}

func TestSafeJoin(t *testing.T) {
	dir := t.TempDir()
	if _, err := SafeJoin(dir, "bgm.mp3"); err != nil {
		t.Errorf("SafeJoin 正常文件报错: %v", err)
	}
	for _, evil := range []string{"../../etc/passwd", "..", "sub/b.mp3"} {
		if _, err := SafeJoin(dir, evil); err == nil {
			t.Errorf("SafeJoin(%q) 应报错", evil)
		}
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"bgm.mp3", "b.wav", "note.txt", "a.jpg"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := List(dir)
	if err != nil {
		t.Fatalf("List 报错: %v", err)
	}
	want := []string{"b.wav", "bgm.mp3"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("List = %v, 期望 %v", got, want)
	}
}
