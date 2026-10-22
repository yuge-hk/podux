package credentials

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTemporaryTokenPermissionsAndExclusiveCreation(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "private-runtime")
	if e := PrivateDir(dir); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(dir, "token")
	if e := PrivateFile(path, []byte("synthetic-test-token")); e != nil {
		t.Fatal(e)
	}
	info, e := os.Lstat(path)
	if e != nil {
		t.Fatal(e)
	}
	if e = checkPermissions(path, info); e != nil {
		t.Fatal("temporary credential access control failed")
	}
	if e = PrivateFile(path, []byte("replacement")); e == nil {
		t.Fatal("existing runtime credential overwritten")
	}
	data, e := os.ReadFile(path)
	if e != nil || string(data) != "synthetic-test-token" {
		t.Fatal("credential contents changed")
	}
}
