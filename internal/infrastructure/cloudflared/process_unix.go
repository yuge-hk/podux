//go:build darwin || linux

package cloudflared

import (
	"os/exec"
	"syscall"
)

func signalProcess(c *exec.Cmd) { _ = syscall.Kill(-c.Process.Pid, syscall.SIGTERM) }
func killProcess(c *exec.Cmd)   { _ = syscall.Kill(-c.Process.Pid, syscall.SIGKILL) }
