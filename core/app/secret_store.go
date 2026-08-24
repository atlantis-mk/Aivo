package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type SecretStore interface {
	Put(context.Context, string, string) error
	Get(context.Context, string) (string, error)
	Delete(context.Context, string) error
}

type LocalSecretStore struct {
	path string
	mu   sync.Mutex
}

func NewDefaultSecretStore() SecretStore {
	home, err := os.UserHomeDir()
	if err != nil {
		return NewMemorySecretStore()
	}
	return &LocalSecretStore{path: filepath.Join(home, ".aivo", "secrets.json")}
}

func NewLocalSecretStore(path string) *LocalSecretStore {
	return &LocalSecretStore{path: path}
}

func (s *LocalSecretStore) Put(ctx context.Context, ref string, value string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return errors.New("secret reference is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.read()
	if err != nil {
		return err
	}
	data[ref] = value
	return s.write(data)
}

func (s *LocalSecretStore) Get(ctx context.Context, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", errors.New("secret reference is required")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.read()
	if err != nil {
		return "", err
	}
	return data[ref], nil
}

func (s *LocalSecretStore) Delete(ctx context.Context, ref string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.read()
	if err != nil {
		return err
	}
	delete(data, ref)
	return s.write(data)
}

func (s *LocalSecretStore) read() (map[string]string, error) {
	if s == nil || strings.TrimSpace(s.path) == "" {
		return map[string]string{}, nil
	}
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var data map[string]string
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	if data == nil {
		data = map[string]string{}
	}
	return data, nil
}

func (s *LocalSecretStore) write(data map[string]string) error {
	if s == nil || strings.TrimSpace(s.path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Chmod(s.path, 0o600)
}

type MemorySecretStore struct {
	mu   sync.Mutex
	data map[string]string
}

func NewMemorySecretStore() *MemorySecretStore {
	return &MemorySecretStore{data: map[string]string{}}
}

func (s *MemorySecretStore) Put(ctx context.Context, ref string, value string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data == nil {
		s.data = map[string]string{}
	}
	s.data[strings.TrimSpace(ref)] = value
	return nil
}

func (s *MemorySecretStore) Get(ctx context.Context, ref string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data[strings.TrimSpace(ref)], nil
}

func (s *MemorySecretStore) Delete(ctx context.Context, ref string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, strings.TrimSpace(ref))
	return nil
}
