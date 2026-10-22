//go:build !linux && !darwin && !windows

package cloudflared

import (
	"os"
	"os/exec"
)

func prepareProcess(c *exec.Cmd) {}
func signalProcess(c *exec.Cmd)  { _ = c.Process.Signal(os.Interrupt) }
func killProcess(c *exec.Cmd)    { _ = c.Process.Kill() }
