package persistence

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	sqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

const sqliteConnectionPragmas = "_pragma=busy_timeout(5000)"

type Store struct {
	db                   *gorm.DB
	sqlDB                *sql.DB
	path                 string
	managedExtensionRoot string
	managedPromptRoot    string
	projectBindingMu     sync.Map
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
	managedRoot, err := defaultManagedExtensionRoot()
	if err != nil {
		return nil, err
	}
	promptRoot, err := defaultManagedPromptRoot()
	if err != nil {
		return nil, err
	}
	return openStore(filepath.Join(dir, "aivo.db"), managedRoot, promptRoot)
}

func Open(path string) (*Store, error) {
	return openStore(path, "", "")
}

func openStore(path, managedExtensionRoot, managedPromptRoot string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := gorm.Open(sqlite.Open(sqliteDSN(path)), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	store := &Store{db: db, sqlDB: sqlDB, path: path, managedExtensionRoot: managedExtensionRoot, managedPromptRoot: managedPromptRoot}
	if err := store.migrate(context.Background()); err != nil {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}

// sqliteDSN configures every pooled SQLite connection. The busy timeout turns
// short-lived competing writes (for example OAuth completion and model refresh)
// into a wait instead of an immediate SQLITE_BUSY failure.
func sqliteDSN(path string) string {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return path + separator + sqliteConnectionPragmas
}

func (s *Store) ManagedPromptRoot() (string, error) {
	if s == nil || s.path == "" || s.path == ":memory:" {
		return "", errors.New("managed prompt storage requires a persistent database path")
	}
	root := s.managedPromptRoot
	if root == "" {
		root = filepath.Join(filepath.Dir(s.path), "prompts")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", err
	}
	if err := os.Chmod(root, 0o700); err != nil {
		return "", err
	}
	return root, nil
}

func (s *Store) Close() error {
	if s == nil || s.sqlDB == nil {
		return nil
	}
	return s.sqlDB.Close()
}

func (s *Store) ManagedExtensionRoot() (string, error) {
	if s == nil || s.path == "" || s.path == ":memory:" {
		return "", errors.New("managed extension storage requires a persistent database path")
	}
	root := s.managedExtensionRoot
	if root == "" {
		root = filepath.Join(filepath.Dir(s.path), "extensions")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", err
	}
	if err := os.Chmod(root, 0o700); err != nil {
		return "", err
	}
	return root, nil
}

func (s *Store) LegacyManagedExtensionRoot() (string, error) {
	if s == nil || s.path == "" || s.path == ":memory:" {
		return "", errors.New("legacy managed extension storage requires a persistent database path")
	}
	return filepath.Abs(filepath.Join(filepath.Dir(s.path), "extensions"))
}

func defaultManagedExtensionRoot() (string, error) {
	configRoot, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configRoot, "Aivo", "Default", "Extensions"), nil
}

func defaultManagedPromptRoot() (string, error) {
	configRoot, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configRoot, "Aivo", "Default", "Prompts"), nil
}
