// Package filestore provides durable single-writer native development saves.
package filestore

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

type Store struct{ Path string }

func (s Store) Load() ([]byte, error) {
	f, e := os.Open(s.Path)
	if errors.Is(e, os.ErrNotExist) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, 1024*1024+1))
	if len(b) > 1024*1024 {
		return nil, errors.New("snapshot too large")
	}
	return b, e
}
func (s Store) Commit(b []byte) error {
	dir := filepath.Dir(s.Path)
	if e := os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	f, e := os.CreateTemp(dir, ".rail-*.tmp")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	if e = os.Rename(f.Name(), s.Path); e != nil {
		return e
	}
	// A rename is the commit point. Directory sync is best-effort on platforms
	// which do not support it; never report rollback after successful promotion.
	if d, e := os.Open(dir); e == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}
