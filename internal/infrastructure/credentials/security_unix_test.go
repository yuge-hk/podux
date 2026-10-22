//go:build linux || darwin

package credentials

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInsecureKeyAndSymlinkBoundary(t *testing.T) {
	dir := t.TempDir()
	data := filepath.Join(dir, "data")
	os.MkdirAll(data, 0700)
	key := filepath.Join(dir, "key")
	os.WriteFile(key, make([]byte, 32), 0644)
	if _, e := Open(data, key, true); e == nil {
		t.Fatal("world-readable key accepted")
	}
	os.Remove(key)
	link := filepath.Join(dir, "link")
	if e := os.Symlink(data, link); e != nil {
		t.Fatal(e)
	}
	if _, e := Open(data, filepath.Join(link, "master.key"), false); e == nil {
		t.Fatal("key allowed through symlink into data")
	}
}
