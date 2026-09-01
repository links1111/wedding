package handler

import (
	"strings"
	"testing"
)

func TestValidateGuestInput(t *testing.T) {
	valid := []guestInput{
		{Name: "张三", Phone: "13800000000", Attending: 1, Headcount: 2, Message: "你好"},
		{Name: "  李四  ", Phone: "", Attending: 0, Headcount: 0, Message: ""},
		{Name: "王五", Phone: "123", Attending: 2, Headcount: 20, Message: strings.Repeat("x", 500)},
	}
	for i, in := range valid {
		if err := validateGuestInput(&in); err != nil {
			t.Errorf("用例%d 应通过: %v", i, err)
		}
	}

	// 副作用：trim 姓名、人数 0→1、超长留言截断
	v := guestInput{Name: "  李四  ", Headcount: 0, Message: strings.Repeat("x", 600)}
	if err := validateGuestInput(&v); err != nil {
		t.Fatalf("合法输入报错: %v", err)
	}
	if v.Name != "李四" {
		t.Errorf("姓名未 trim: %q", v.Name)
	}
	if v.Headcount != 1 {
		t.Errorf("人数未默认 1: %d", v.Headcount)
	}
	if len(v.Message) != 500 {
		t.Errorf("留言未截断为 500: %d", len(v.Message))
	}

	invalid := []guestInput{
		{Name: "", Phone: "", Attending: 1, Headcount: 1},
		{Name: strings.Repeat("张", 51), Phone: "", Attending: 1, Headcount: 1},
		{Name: "张三", Phone: strings.Repeat("1", 21), Attending: 1, Headcount: 1},
		{Name: "张三", Phone: "", Attending: 3, Headcount: 1},
		{Name: "张三", Phone: "", Attending: -1, Headcount: 1},
	}
	for i, in := range invalid {
		if err := validateGuestInput(&in); err == nil {
			t.Errorf("用例%d 应报错", i)
		}
	}
}
