//go:build !windows

package main

import (
	"errors"
	"os/exec"
)

func runGUI(options updateOptions) error { return errors.New("the updater window requires Windows") }
func prepareCLI()                        {}

func hideConsole(command *exec.Cmd) {}
