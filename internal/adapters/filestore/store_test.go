package filestore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCommitReloadAndExclusiveOwner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.json")
	unlock, e := Lock(path)
	if e != nil {
		t.Fatal(e)
	}
	if other, e := Lock(path); e == nil {
		other()
		t.Fatal("two writers acquired same save")
	}
	s := Store{Path: path}
	if e = s.Commit([]byte("first")); e != nil {
		t.Fatal(e)
	}
	if e = s.Commit([]byte("second")); e != nil {
		t.Fatal(e)
	}
	b, e := s.Load()
	if e != nil || string(b) != "second" {
		t.Fatal(string(b), e)
	}
	if files, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".rail-*.tmp")); len(files) != 0 {
		t.Fatal("temporary save leaked")
	}
	unlock()
	unlock, e = Lock(path)
	if e != nil {
		t.Fatal(e)
	}
	unlock()
	if e = os.WriteFile(path, make([]byte, 1024*1024+1), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Load(); e == nil {
		t.Fatal("unbounded snapshot accepted")
	}
}
