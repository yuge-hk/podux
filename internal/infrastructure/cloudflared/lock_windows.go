package cloudflared

import (
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	d "podux/internal/domain/tunnel"
)

func LockDataDir(dataDir string) (func(), error) {
	f, e := os.OpenFile(filepath.Join(dataDir, "cloudflared.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, d.ErrConflict
	}
	o := &windows.Overlapped{}
	if e = windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, o); e != nil {
		f.Close()
		return nil, d.ErrConflict
	}
	return func() { _ = windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, o); _ = f.Close() }, nil
}
