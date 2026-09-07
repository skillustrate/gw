package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/gitskill/gw/internal/cliapp"
	"github.com/gitskill/gw/internal/exec"
)

func TestMain_Help(t *testing.T) {
	var stdout, stderr bytes.Buffer
	app := cliapp.New(&stdout, &stderr, exec.NewFakeRunner(nil))
	code := app.Run(context.Background(), []string{"--help"})
	if code != 0 {
		t.Errorf("expected exit code 0 for --help, got %d", code)
	}
}