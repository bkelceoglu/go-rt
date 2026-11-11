package plugin

import (
	"context"

	"github.com/caveman/go-rt/internal/config"
	"github.com/caveman/go-rt/internal/domain"
	"github.com/caveman/go-rt/internal/store"
)

// All returns the complete sorted catalog of built-in redteam plugins from SQLite.
func All(ctx context.Context, db *store.SQLiteStore) ([]store.PluginRecord, error) {
	return db.ListPlugins(ctx)
}

// BuildCatalogSnapshot constructs a CatalogSnapshot from SQLite for config validation and auto-fix.
func BuildCatalogSnapshot(ctx context.Context, db *store.SQLiteStore) *config.CatalogSnapshot {
	if db == nil {
		return nil
	}
	snap := &config.CatalogSnapshot{
		Plugins:     make(map[string]bool),
		Collections: make(map[string]bool),
		Strategies:  make(map[string]bool),
	}
	populateSnapshotMaps(ctx, db, snap)
	return snap
}

func populateSnapshotMaps(ctx context.Context, db *store.SQLiteStore, snap *config.CatalogSnapshot) {
	if plugins, err := db.ListPlugins(ctx); err == nil {
		for _, p := range plugins {
			snap.Plugins[p.ID] = true
		}
	}
	if cols, err := db.CollectionMap(ctx); err == nil {
		for id := range cols {
			snap.Collections[id] = true
		}
	}
	if strats, err := db.ListStrategies(ctx); err == nil {
		for _, s := range strats {
			snap.Strategies[s.ID] = true
		}
	}
}

// ExpandSpecs expands any gort:collection:* entries into individual PluginSpecs using SQLite collections.
func ExpandSpecs(ctx context.Context, db *store.SQLiteStore, specs []domain.PluginSpec, defaultNumTests int) ([]domain.PluginSpec, error) {
	collections, err := db.CollectionMap(ctx)
	if err != nil {
		return nil, err
	}
	exp := &specExpander{
		collections: collections,
		seen:        make(map[string]bool),
		defaultNum:  defaultNumTests,
		out:         []domain.PluginSpec{},
	}
	for _, spec := range specs {
		exp.appendSpec(spec)
	}
	return exp.out, nil
}

type specExpander struct {
	collections map[string][]string
	seen        map[string]bool
	defaultNum  int
	out         []domain.PluginSpec
}

func (e *specExpander) appendSpec(spec domain.PluginSpec) {
	if spec.NumTests <= 0 {
		spec.NumTests = e.defaultNum
	}
	if members, ok := e.collections[spec.ID]; ok {
		e.appendCollectionMembers(members, spec)
		return
	}
	if !e.seen[spec.ID] {
		e.seen[spec.ID] = true
		e.out = append(e.out, spec)
	}
}

func (e *specExpander) appendCollectionMembers(members []string, parent domain.PluginSpec) {
	for _, id := range members {
		if !e.seen[id] {
			e.seen[id] = true
			e.out = append(e.out, domain.PluginSpec{
				ID:       id,
				NumTests: parent.NumTests,
				Severity: parent.Severity,
				Config:   parent.Config,
			})
		}
	}
}
