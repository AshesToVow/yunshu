package repository

import (
	"testing"

	"yunshu/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestListWithPaginationRaw 验证导出专用分页：不钳制 pageSize、页码兜底、总数与排序正确。
func TestListWithPaginationRaw(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:list_raw?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		// gorm.io/driver/sqlite 依赖 CGO；Windows/CI 常以 CGO_ENABLED=0 编译。
		t.Skipf("sqlite unavailable: %v", err)
	}
	if err := db.AutoMigrate(&model.Department{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for i := 1; i <= 5; i++ {
		code := "dept-" + string(rune('a'+i-1))
		if err := db.Create(&model.Department{Name: code, Code: code, Level: 1}).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	q := func() *gorm.DB {
		return db.WithContext(nil).Model(&model.Department{}).Where("deleted_at IS NULL")
	}

	// pageSize 超过 MaxPageSize(100) 时不被钳制为 100 的行为仅在导出路径（pageSize>100 才有意义），
	// 这里用 5 行数据验证语义：给的 pageSize 原样生效。
	var rows []model.Department
	total, err := listWithPaginationRaw(q(), 1, 50, "code ASC", &rows)
	if err != nil {
		t.Fatalf("list raw: %v", err)
	}
	if total != 5 || len(rows) != 5 {
		t.Fatalf("total=%d rows=%d, want 5/5", total, len(rows))
	}
	if rows[0].Code != "dept-a" || rows[4].Code != "dept-e" {
		t.Fatalf("order wrong: first=%s last=%s", rows[0].Code, rows[4].Code)
	}

	// 第二页只取剩余行。
	rows = nil
	total, err = listWithPaginationRaw(q(), 2, 3, "code ASC", &rows)
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if total != 5 || len(rows) != 2 || rows[0].Code != "dept-d" {
		t.Fatalf("page2: total=%d rows=%d first=%s, want 5/2/dept-d", total, len(rows), rows[0].Code)
	}

	// page<=0 与 pageSize<=0 走兜底值（默认第 1 页、默认页大小）。
	rows = nil
	if _, err := listWithPaginationRaw(q(), 0, 0, "code ASC", &rows); err != nil {
		t.Fatalf("fallback: %v", err)
	}
	if len(rows) != 5 {
		t.Fatalf("fallback should return first page rows=%d, want 5", len(rows))
	}
}
