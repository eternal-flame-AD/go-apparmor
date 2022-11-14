package apparmor

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func shell(cmdline string, args ...string) (string, error) {
	args = append([]string{"-c", cmdline}, args...)
	cmd := exec.Command("sh", args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func poll(timeout time.Duration, fn func() (bool, error)) error {
	deadline := time.Now().Add(timeout)
	for {
		done, err := fn()
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out")
		}
		time.Sleep(time.Millisecond * 100)
	}
}
