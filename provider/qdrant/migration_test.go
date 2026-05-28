package qdrant

import "testing"

func TestNormalizeMigrationsSortsByVersion(t *testing.T) {
	t.Parallel()

	migrations := []Migration{
		{Version: 2, Name: "002_add_indexes"},
		{Version: 1, Name: "001_init"},
	}

	normalized, err := normalizeMigrations(migrations)
	if err != nil {
		t.Fatalf("normalizeMigrations returned error: %v", err)
	}
	if normalized[0].Version != 1 || normalized[1].Version != 2 {
		t.Fatalf("unexpected order: %#v", normalized)
	}
}

func TestNormalizeMigrationsRejectsDuplicates(t *testing.T) {
	t.Parallel()

	_, err := normalizeMigrations([]Migration{
		{Version: 1, Name: "001_init"},
		{Version: 1, Name: "001_duplicate"},
	})
	if err == nil {
		t.Fatal("expected duplicate version error, got nil")
	}
}
