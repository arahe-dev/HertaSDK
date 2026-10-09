package mutate

import (
	"os/exec"
	"strconv"
)

func ownGroup(*exec.Cmd) {}

// killTree ends cmd and every descendant; killing go alone would orphan
// the test binary, which keeps the workspace's files open.
func killTree(cmd *exec.Cmd) error {
	return exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
}
