package credentials

import (
	"os"
	d "podux/internal/domain/tunnel"
)

// PrivateDir is reserved for a newly created application-owned runtime folder.
func PrivateDir(path string) error {
	if e := os.MkdirAll(path, 0700); e != nil {
		return d.ErrCredentials
	}
	i, e := os.Lstat(path)
	if e != nil || !i.IsDir() || i.Mode()&os.ModeSymlink != 0 {
		return d.ErrCredentials
	}
	if protectDirectory(path) != nil {
		return d.ErrCredentials
	}
	return nil
}
func PrivateFile(path string, content []byte) error {
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return d.ErrCredentials
	}
	defer f.Close()
	if protectFile(path) != nil {
		return d.ErrCredentials
	}
	if _, e = f.Write(content); e != nil {
		return d.ErrCredentials
	}
	return f.Sync()
}
