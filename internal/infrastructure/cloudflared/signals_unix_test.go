//go:build linux || darwin

package cloudflared

import (
	"os"
	"os/signal"
	"syscall"
)

func ignoreTerm()                 { signal.Ignore(syscall.SIGTERM) }
func notifyTerm(c chan os.Signal) { signal.Notify(c, syscall.SIGTERM) }
