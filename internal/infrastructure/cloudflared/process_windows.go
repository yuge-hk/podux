package cloudflared

import (
	"golang.org/x/sys/windows"
	"os/exec"
	"syscall"
)

func prepareProcess(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP}
}
func signalProcess(c *exec.Cmd) {
	_ = windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(c.Process.Pid))
}
func killProcess(c *exec.Cmd) { _ = c.Process.Kill() }
