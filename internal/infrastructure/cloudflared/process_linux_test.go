package cloudflared

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Tests the Linux-specific parent-death mechanism independently of the portable
// lifecycle tests. No equivalent guarantee is claimed on macOS/Windows.
func TestLinuxParentHelper(t *testing.T) {
	dir := ""
	for j, a := range os.Args {
		if a == "--parent-helper" && j+1 < len(os.Args) {
			dir = os.Args[j+1]
		}
	}
	if dir == "" {
		return
	}
	c := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$", "--", "--helper", "ignore", "--metrics", "127.0.0.1:0", "--token-file", filepath.Join(dir, "token"))
	prepareProcess(c)
	if e := c.Start(); e != nil {
		os.Exit(12)
	}
	fmt.Println(c.Process.Pid)
	select {}
}
func TestLinuxParentDeath(t *testing.T) {
	dir := t.TempDir()
	if e := os.WriteFile(filepath.Join(dir, "token"), []byte("test-only"), 0600); e != nil {
		t.Fatal(e)
	}
	c := exec.Command(os.Args[0], "-test.run=^TestLinuxParentHelper$", "--", "--parent-helper", dir)
	out, e := c.StdoutPipe()
	if e != nil {
		t.Fatal(e)
	}
	if e = c.Start(); e != nil {
		t.Fatal(e)
	}
	defer func() { _ = c.Process.Kill(); _ = c.Wait() }()
	scan := bufio.NewScanner(out)
	if !scan.Scan() {
		t.Fatal("no child pid")
	}
	pid, e := strconv.Atoi(scan.Text())
	if e != nil {
		t.Fatal(e)
	}
	defer syscall.Kill(pid, syscall.SIGKILL)
	_ = c.Process.Kill()
	_ = c.Wait()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		raw, e := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if os.IsNotExist(e) {
			return
		}
		if e == nil {
			end := strings.LastIndex(string(raw), ")")
			if end >= 0 && strings.HasPrefix(string(raw)[end+1:], " Z") {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("child remained running after parent death")
}
