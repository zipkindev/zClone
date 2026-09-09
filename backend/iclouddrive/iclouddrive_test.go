//go:build !plan9 && !solaris

package iclouddrive_test

import (
	"testing"

	"zclone/backend/iclouddrive"
	"zclone/fstest/fstests"
)

// TestIntegration runs integration tests against the remote
func TestIntegration(t *testing.T) {
	fstests.Run(t, &fstests.Opt{
		RemoteName: "TestICloudDrive:",
		NilObject:  (*iclouddrive.Object)(nil),
	})
}
