//go:build !linux && (!darwin || !amd64)

// Package mount2 implements a FUSE mounting system for zclone remotes.
//
// Build for mount for unsupported platforms to stop go complaining
// about "no buildable Go source files".
package mount2
