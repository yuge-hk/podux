//go:build !windows

package credentials

import (
	"fmt"
	"os"
)

func checkPermissions(_ string, i os.FileInfo) error {
	if i.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("insecure permissions")
	}
	return nil
}
func protectFile(path string) error      { return os.Chmod(path, 0600) }
func protectDirectory(path string) error { return os.Chmod(path, 0700) }
