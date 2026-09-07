package main

import (
	"context"
	"os"

	"github.com/gitskill/gw/internal/cliapp"
	"github.com/gitskill/gw/internal/exec"
)

func main() {
	runner := exec.RealRunner{}
	app := cliapp.New(os.Stdout, os.Stderr, runner)
	code := app.Run(context.Background(), os.Args[1:])
	os.Exit(code)
}