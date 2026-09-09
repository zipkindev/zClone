// Test Gofile filesystem interface
package gofile_test

import (
	"testing"

	"zclone/backend/gofile"
	"zclone/fstest/fstests"
)

// TestIntegration runs integration tests against the remote
func TestIntegration(t *testing.T) {
	fstests.Run(t, &fstests.Opt{
		RemoteName: "TestGoFile:",
		NilObject:  (*gofile.Object)(nil),
	})
}
