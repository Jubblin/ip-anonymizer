package mapping

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	version    = 1
	maxAlloc   = 254
	fakePrefix = "198.51.100."
)

type fileData struct {
	Version  int               `json:"version"`
	Next     int               `json:"next"`
	Mappings map[string]string `json:"mappings"`
}

type Store struct {
	path     string
	next     int
	mappings map[string]string
	dirty    bool
}

func Load(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("create mapping directory: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read mapping file: %w", err)
		}

		store := &Store{
			path:     path,
			next:     1,
			mappings: make(map[string]string),
		}
		if err := store.Save(); err != nil {
			return nil, err
		}
		return store, nil
	}

	var fd fileData
	if err := json.Unmarshal(data, &fd); err != nil {
		return nil, fmt.Errorf("parse mapping file: %w", err)
	}

	if fd.Mappings == nil {
		fd.Mappings = make(map[string]string)
	}
	if fd.Next < 1 {
		fd.Next = 1
	}

	return &Store{
		path:     path,
		next:     fd.Next,
		mappings: fd.Mappings,
	}, nil
}

func (s *Store) Lookup(real string) (string, bool) {
	fake, ok := s.mappings[real]
	return fake, ok
}

func (s *Store) Allocate(real string) (string, error) {
	if fake, ok := s.mappings[real]; ok {
		return fake, nil
	}

	if s.next > maxAlloc {
		return "", fmt.Errorf("exhausted fake IP pool (max %d addresses in %s0/24)", maxAlloc, fakePrefix)
	}

	fake := fmt.Sprintf("%s%d", fakePrefix, s.next)
	s.mappings[real] = fake
	s.next++
	s.dirty = true
	return fake, nil
}

func (s *Store) Save() error {
	if !s.dirty && fileExists(s.path) {
		return nil
	}

	fd := fileData{
		Version:  version,
		Next:     s.next,
		Mappings: s.mappings,
	}

	payload, err := json.MarshalIndent(fd, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal mapping file: %w", err)
	}
	payload = append(payload, '\n')

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o600); err != nil {
		return fmt.Errorf("write mapping temp file: %w", err)
	}

	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename mapping file: %w", err)
	}

	if err := os.Chmod(s.path, 0o600); err != nil {
		return fmt.Errorf("chmod mapping file: %w", err)
	}

	s.dirty = false
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
