package credentials

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVault(t *testing.T) {
	dir := t.TempDir()
	data := filepath.Join(dir, "data")
	key := filepath.Join(dir, "secrets", "master.key")
	v, e := Open(data, key, false)
	if e != nil {
		t.Fatal(e)
	}
	a, e := v.Encrypt("abc", "test-secret")
	if e != nil {
		t.Fatal(e)
	}
	b, _ := v.Encrypt("abc", "test-secret")
	if a == b || strings.Contains(a, "test-secret") {
		t.Fatal("encryption")
	}
	if p, e := v.Decrypt("abc", a); e != nil || p != "test-secret" {
		t.Fatal("decrypt")
	}
	if _, e = v.Decrypt("other", a); e == nil {
		t.Fatal("AAD")
	}
	if _, e = v.Decrypt("abc", a[:len(a)-3]); e == nil {
		t.Fatal("tamper")
	}
	v2, e := Open(data, key, true)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = v2.Decrypt("abc", a); e != nil {
		t.Fatal(e)
	}
	if e = os.Remove(key); e != nil {
		t.Fatal(e)
	}
	if _, e = Open(data, key, true); e == nil {
		t.Fatal("must not recreate key")
	}
	if _, e = os.Stat(key); !os.IsNotExist(e) {
		t.Fatal("replacement key created")
	}
	if _, e = Open(data, filepath.Join(data, "key"), false); e == nil {
		t.Fatal("key inside data directory")
	}
}
func TestWrongKeyIsNeverReplaced(t *testing.T) {
	dir := t.TempDir()
	data := filepath.Join(dir, "data")
	key := filepath.Join(dir, "master.key")
	v, e := Open(data, key, false)
	if e != nil {
		t.Fatal(e)
	}
	value, _ := v.Encrypt("id", "synthetic-secret")
	replacement := []byte(strings.Repeat("x", 32))
	if e = os.WriteFile(key, replacement, 0600); e != nil {
		t.Fatal(e)
	}
	other, e := Open(data, key, true)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = other.Decrypt("id", value); e == nil {
		t.Fatal("accepted wrong key")
	}
	after, _ := os.ReadFile(key)
	if string(after) != string(replacement) {
		t.Fatal("silently replaced key")
	}
}
