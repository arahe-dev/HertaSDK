//go:build unix

package mutate

import (
	"os/exec"
	"syscall"
)

// ownGroup puts cmd in a fresh process group so killTree reaches the test
// binary that go test spawns.
func ownGroup(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }

func killTree(cmd *exec.Cmd) error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
