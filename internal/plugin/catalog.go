package plugin

import (
	"context"

	"github.com/caveman/go-rt/internal/domain"
	"github.com/caveman/go-rt/internal/store"
)

// All returns the complete sorted catalog of built-in redteam plugins from SQLite.
func All(ctx context.Context, db *store.SQLiteStore) ([]store.PluginRecord, error) {
	return db.ListPlugins(ctx)
}

// ExpandSpecs expands any gort:collection:* entries into individual PluginSpecs using SQLite collections.
func ExpandSpecs(ctx context.Context, db *store.SQLiteStore, specs []domain.PluginSpec, defaultNumTests int) ([]domain.PluginSpec, error) {
	collections, err := db.CollectionMap(ctx)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var out []domain.PluginSpec
	for _, spec := range specs {
		out = appendExpandedSpec(out, seen, collections, spec, defaultNumTests)
	}
	return out, nil
}

func appendExpandedSpec(
	out []domain.PluginSpec,
	seen map[string]bool,
	collections map[string][]string,
	spec domain.PluginSpec,
	defaultNum int,
) []domain.PluginSpec {
	num := max(spec.NumTests, defaultNum)
	if members, ok := collections[spec.ID]; ok {
		for _, id := range members {
			if !seen[id] {
				seen[id] = true
				out = append(out, domain.PluginSpec{ID: id, NumTests: num, Severity: spec.Severity})
			}
		}
		return out
	}
	if !seen[spec.ID] {
		seen[spec.ID] = true
		spec.NumTests = num
		out = append(out, spec)
	}
	return out
}
