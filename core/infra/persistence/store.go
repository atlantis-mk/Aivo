package persistence

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"

	sqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type Store struct {
	db    *gorm.DB
	sqlDB *sql.DB
}

func OpenDefault() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".aivo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return Open(filepath.Join(dir, "aivo.db"))
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	store := &Store{db: db, sqlDB: sqlDB}
	if err := store.migrate(context.Background()); err != nil {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.sqlDB == nil {
		return nil
	}
	return s.sqlDB.Close()
}
