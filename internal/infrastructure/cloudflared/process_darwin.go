package cloudflared

import (
	"os/exec"
	"syscall"
)

func prepareProcess(c *exec.Cmd) { c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }
