//go:build !windows

package main

import "errors"

func runGUI(options updateOptions) error { return errors.New("the updater window requires Windows") }
func prepareCLI()                        {}
