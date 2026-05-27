// parity_exec_test.go provides low-level exec helpers for the parity harness.
package main

import (
	"bytes"
	"os/exec"
	"testing"
)

// runGoInDirBin runs an arbitrary binary in a specific directory.
// This is the low-level helper used by runGoInDirWith and runPyInDir.
func runGoInDirBin(t *testing.T, dir, bin string, args ...string) (string, string, int) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			// Non-ExitError means the binary couldn't be launched.
			return "", "", -1
		}
	}
	return outBuf.String(), errBuf.String(), code
}
