package protondrive_test

import (
	"testing"

	"zclone/backend/protondrive"
	"zclone/fstest/fstests"
)

// TestIntegration runs integration tests against the remote
func TestIntegration(t *testing.T) {
	fstests.Run(t, &fstests.Opt{
		RemoteName: "TestProtonDrive:",
		NilObject:  (*protondrive.Object)(nil),
	})
}
