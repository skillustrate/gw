package exec

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"
)

// Result captures the output and status of an external command execution.
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// Runner provides a safe command execution boundary.
// Commands are executed with exact argument vectors — never via shell string interpolation.
type Runner interface {
	Run(ctx context.Context, dir, name string, args ...string) (Result, error)
}

// RealRunner executes external system binaries using os/exec.
type RealRunner struct {
	Timeout time.Duration
}

// Run executes the named binary with args in the specified working directory.
func (r RealRunner) Run(ctx context.Context, dir, name string, args ...string) (Result, error) {
	if r.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	stdout := stdoutBuf.String()
	stderr := stderrBuf.String()

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return Result{
				ExitCode: exitErr.ExitCode(),
				Stdout:   stdout,
				Stderr:   stderr,
			}, nil
		}
		return Result{
			ExitCode: -1,
			Stdout:   stdout,
			Stderr:   stderr,
		}, err
	}

	return Result{
		ExitCode: 0,
		Stdout:   stdout,
		Stderr:   stderr,
	}, nil
}