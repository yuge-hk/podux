package cloudflared

import (
	"os"
	"path/filepath"
	d "podux/internal/domain/tunnel"
	"podux/internal/infrastructure/credentials"
	"regexp"
)

var runtimeName = regexp.MustCompile(`^[a-z0-9]{15}-[0-9]+$`)

// PrepareRuntime must run only while holding LockDataDir, before starting any
// managed child. It removes stale plaintext credential files after a crash.
func PrepareRuntime(dataDir string) error {
	root := filepath.Join(dataDir, "cloudflared")
	if e := credentials.PrivateDir(root); e != nil {
		return d.ErrCredentials
	}
	entries, e := os.ReadDir(root)
	if e != nil {
		return d.ErrCredentials
	}
	for _, entry := range entries {
		if entry.IsDir() && runtimeName.MatchString(entry.Name()) {
			if e = os.RemoveAll(filepath.Join(root, entry.Name())); e != nil {
				return d.ErrCredentials
			}
		}
	}
	return nil
}
