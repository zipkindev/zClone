//go:build !windows

package main

import "errors"

func allocatedBytes(name string) (uint64, error) {
	return 0, errors.New("native allocation checks require Windows")
}
