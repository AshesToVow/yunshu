package bootstrap

import (
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSchemaVersionRoundTrip(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		// gorm.io/driver/sqlite 依赖 CGO；Windows/CI 常以 CGO_ENABLED=0 编译。
		if strings.Contains(err.Error(), "CGO") || strings.Contains(err.Error(), "requires cgo") {
			t.Skip(err.Error())
		}
		t.Fatal(err)
	}
	if err := CheckSchemaVersion(db); err == nil {
		t.Fatal("expected missing version error")
	}
	if err := RecordSchemaVersion(db); err != nil {
		t.Fatal(err)
	}
	if err := CheckSchemaVersion(db); err != nil {
		t.Fatal(err)
	}
}

func TestCheckSchemaVersionNilDB(t *testing.T) {
	if err := CheckSchemaVersion(nil); err == nil {
		t.Fatal("expected nil db error")
	}
}
