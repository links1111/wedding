package models

import "testing"

func TestAutoMigrateIncludesWeddingAndIsolationFields(t *testing.T) {
	// 占位：确保结构体字段存在（编译期验证）
	_ = Wedding{Token: "x", Name: "y"}
	_ = Setting{WeddingID: 1, Key: "k", Value: "v"}
	_ = Guest{WeddingID: 1}
	_ = Visit{WeddingID: 1}
}
