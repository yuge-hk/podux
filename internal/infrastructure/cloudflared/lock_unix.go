//go:build linux || darwin

package cloudflared

import (
	"golang.org/x/sys/unix"
	"os"
	d "podux/internal/domain/tunnel"
)

// LockDataDir prevents two managers sharing one PocketBase data directory.
// Keep the descriptor open; never unlink a lock file while another process may
// hold it, since replacing its inode defeats mutual exclusion.
func LockDataDir(dataDir string) (func(), error) {
	f, e := os.OpenFile(dataDir+"/cloudflared.lock", os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, d.ErrConflict
	}
	if e = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); e != nil {
		f.Close()
		return nil, d.ErrConflict
	}
	return func() { _ = unix.Flock(int(f.Fd()), unix.LOCK_UN); _ = f.Close() }, nil
}
