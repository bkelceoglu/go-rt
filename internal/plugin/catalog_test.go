package plugin

import (
	"path/filepath"
	"testing"

	"github.com/bkelceoglu/go-rt/internal/domain"
	"github.com/bkelceoglu/go-rt/internal/store"
)

func TestAllCatalogCountAndCollectionsFromSQLite(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_gort.db")
	db, err := store.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	defer db.Close()

	ctx := t.Context()
	all, err := All(ctx, db)
	if err != nil || len(all) < 150 {
		t.Fatalf("expected at least 150 SQLite-seeded plugins, got %d (err=%v)", len(all), err)
	}
	expanded, err := ExpandSpecs(ctx, db, []domain.PluginSpec{
		{ID: "gort:collection:owasp-llm-top-10", NumTests: 5},
		{ID: "gort:collection:financial", NumTests: 3},
	}, 5)
	if err != nil || len(expanded) != 30 {
		t.Fatalf("expected 30 expanded plugins from SQLite collections, got %d (err=%v)", len(expanded), err)
	}
}
