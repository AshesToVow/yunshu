package dictconfig

import (
	"context"
	"fmt"
	"testing"
	"time"

	"yunshu/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openPasswordPolicyTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:pwd_policy_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		// gorm.io/driver/sqlite 依赖 CGO；Windows/CI 常以 CGO_ENABLED=0 编译。
		t.Skipf("sqlite unavailable: %v", err)
	}
	if err := db.AutoMigrate(&model.DictEntry{}); err != nil {
		t.Skipf("sqlite migrate failed: %v", err)
	}
	return db
}

func TestResolvePasswordExpiryDays(t *testing.T) {
	ctx := context.Background()
	if got := ResolvePasswordExpiryDays(ctx, nil); got != 90 {
		t.Fatalf("nil db 应回退默认 90，实际 %d", got)
	}

	db := openPasswordPolicyTestDB(t)
	if got := ResolvePasswordExpiryDays(ctx, db); got != 90 {
		t.Fatalf("无字典项应回退默认 90，实际 %d", got)
	}

	dictType := DefaultPasswordPolicyDictTypes().ExpiryDays
	if err := db.Create(&model.DictEntry{DictType: dictType, Label: "过期天数", Value: "7", Sort: 1, Status: 0}).Error; err != nil {
		t.Fatalf("seed disabled: %v", err)
	}
	if got := ResolvePasswordExpiryDays(ctx, db); got != 90 {
		t.Fatalf("停用项不应生效，实际 %d", got)
	}

	enabled := &model.DictEntry{DictType: dictType, Label: "过期天数", Value: "30", Sort: 1, Status: 1}
	if err := db.Create(enabled).Error; err != nil {
		t.Fatalf("seed enabled: %v", err)
	}
	if got := ResolvePasswordExpiryDays(ctx, db); got != 30 {
		t.Fatalf("应读取启用项 30，实际 %d", got)
	}

	if err := db.Model(enabled).Update("value", "abc").Error; err != nil {
		t.Fatalf("update invalid: %v", err)
	}
	if got := ResolvePasswordExpiryDays(ctx, db); got != 90 {
		t.Fatalf("非法值应回退默认 90，实际 %d", got)
	}

	if err := db.Model(enabled).Update("value", "0").Error; err != nil {
		t.Fatalf("update zero: %v", err)
	}
	if got := ResolvePasswordExpiryDays(ctx, db); got != 0 {
		t.Fatalf("0 表示不过期，应原样返回，实际 %d", got)
	}

	// 与全量解析保持同一取数规则（sort 升序、同 sort 取 id 最大）。
	if err := db.Model(enabled).Update("value", "30").Error; err != nil {
		t.Fatalf("restore: %v", err)
	}
	if got := ResolvePasswordExpiryDays(ctx, db); got != ResolvePasswordPolicy(ctx, db).ExpiryDays {
		t.Fatalf("过期天数与全量策略不一致: %d", got)
	}
}
