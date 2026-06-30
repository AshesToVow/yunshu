package mysqlbackup

import (
	"strings"
	"testing"
)

func TestFormatMysqldumpFlags(t *testing.T) {
	flags, err := FormatMysqldumpFlags(DefaultMysqldumpOptionIDs(), "")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--single-transaction", "--quick", "--routines", "--triggers", "--skip-add-drop-table", "--set-gtid-purged=OFF"} {
		if !strings.Contains(flags, want) {
			t.Fatalf("missing %s in %s", want, flags)
		}
	}
}

func TestFormatMysqldumpFlagsDedupeExtra(t *testing.T) {
	flags, err := FormatMysqldumpFlags([]string{"single_transaction"}, "--single-transaction --set-gtid-purged=OFF")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(flags, "--single-transaction") != 1 {
		t.Fatalf("expected one --single-transaction, got %s", flags)
	}
}

func TestFormatMysqldumpFlagsBooleanFalseSyntax(t *testing.T) {
	flags, err := FormatMysqldumpFlags(nil, "--add-drop-table=FALSE --add-drop-database=FALSE")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--add-drop-table=FALSE", "--add-drop-database=FALSE"} {
		if !strings.Contains(flags, want) {
			t.Fatalf("missing %s in %s", want, flags)
		}
	}
}

func TestValidateMysqldumpOptionIDsConflict(t *testing.T) {
	err := ValidateMysqldumpOptionIDs([]string{"single_transaction", "lock_all_tables"})
	if err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestMysqldumpOptionCatalogSize(t *testing.T) {
	t.Parallel()
	if len(MysqldumpOptionCatalog) < 40 {
		t.Fatalf("expected expanded catalog, got %d options", len(MysqldumpOptionCatalog))
	}
	_, err := FormatMysqldumpFlags([]string{"add_drop_table", "default_charset_utf8mb4", "column_statistics_off"}, "")
	if err != nil {
		t.Fatal(err)
	}
}
