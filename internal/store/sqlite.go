package store

import (
	"cmp"
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bkelceoglu/go-rt/internal/domain"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

//go:embed seed_catalog.sql
var seedCatalogSQL string

const sqlitePragmas = `
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 10000;
PRAGMA synchronous = NORMAL;
PRAGMA foreign_keys = ON;
`

var _ domain.ResultStore = (*SQLiteStore)(nil)

// SQLiteStore manages plugins, collections, dataset_sources, prompt_bank, and OTel results via GORM + pure-Go SQLite.
type SQLiteStore struct {
	db    *gorm.DB
	sqlDB *sql.DB
}

// OpenSQLite opens or creates the SQLite database at dbPath, runs GORM AutoMigrate, and seeds the catalog.
func OpenSQLite(dbPath string) (*SQLiteStore, error) {
	dbPath = cmp.Or(dbPath, ".gort/gort.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite dir: %w", err)
	}
	gormDB, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open gorm sqlite %s: %w", dbPath, err)
	}
	return initSQLiteStore(gormDB)
}

func initSQLiteStore(gormDB *gorm.DB) (*SQLiteStore, error) {
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("extract sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)
	s := &SQLiteStore{db: gormDB, sqlDB: sqlDB}
	return s, s.bootstrap()
}

func (s *SQLiteStore) bootstrap() error {
	if _, err := s.sqlDB.Exec(sqlitePragmas); err != nil {
		return fmt.Errorf("exec sqlite pragmas: %w", err)
	}
	if err := s.autoMigrateModels(); err != nil {
		return fmt.Errorf("gorm automigrate: %w", err)
	}
	if _, err := s.sqlDB.Exec(seedCatalogSQL); err != nil {
		return fmt.Errorf("exec seed_catalog.sql: %w", err)
	}
	return nil
}

func (s *SQLiteStore) autoMigrateModels() error {
	return s.db.AutoMigrate(
		&PluginRecord{},
		&PluginCollectionRecord{},
		&StrategyRecord{},
		&DatasetSourceRecord{},
		&PromptBankRecord{},
		&EvalRunModel{},
		&EvalResultModel{},
		&OTelSpanModel{},
	)
}

// Close closes the underlying SQLite connection.
func (s *SQLiteStore) Close() error {
	return s.sqlDB.Close()
}

// ListPlugins returns all plugins stored in the SQLite plugins table ordered by ID.
func (s *SQLiteStore) ListPlugins(ctx context.Context) ([]PluginRecord, error) {
	out := []PluginRecord{}
	if err := s.db.WithContext(ctx).Order("id ASC").Find(&out).Error; err != nil {
		return nil, fmt.Errorf("list plugins: %w", err)
	}
	return out, nil
}

// CollectionMap returns all preset collections mapped to their plugin IDs from SQLite.
func (s *SQLiteStore) CollectionMap(ctx context.Context) (map[string][]string, error) {
	var records []PluginCollectionRecord
	if err := s.db.WithContext(ctx).Order("collection_id ASC, plugin_id ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("query collection map: %w", err)
	}
	out := make(map[string][]string)
	for _, r := range records {
		out[r.CollectionID] = append(out[r.CollectionID], r.PluginID)
	}
	return out, nil
}

// ListDatasetSources returns all enabled dataset sources from SQLite.
func (s *SQLiteStore) ListDatasetSources(ctx context.Context) ([]DatasetSourceRecord, error) {
	out := []DatasetSourceRecord{}
	if err := s.db.WithContext(ctx).Where("enabled = ?", true).Order("dataset_name ASC").Find(&out).Error; err != nil {
		return nil, fmt.Errorf("list dataset sources: %w", err)
	}
	return out, nil
}

// ListStrategies returns all adversarial strategies stored in SQLite.
func (s *SQLiteStore) ListStrategies(ctx context.Context) ([]StrategyRecord, error) {
	out := []StrategyRecord{}
	if err := s.db.WithContext(ctx).Order("id ASC").Find(&out).Error; err != nil {
		return nil, fmt.Errorf("list strategies: %w", err)
	}
	return out, nil
}

// GetStrategy returns a single strategy by ID from SQLite.
func (s *SQLiteStore) GetStrategy(ctx context.Context, id string) (StrategyRecord, error) {
	var out StrategyRecord
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&out).Error; err != nil {
		return StrategyRecord{}, fmt.Errorf("get strategy %s: %w", id, err)
	}
	return out, nil
}
