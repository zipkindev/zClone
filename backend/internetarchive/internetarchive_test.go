// Test internetarchive filesystem interface
package internetarchive_test

import (
	"testing"

	"zclone/backend/internetarchive"
	"zclone/fstest/fstests"
)

// TestIntegration runs integration tests against the remote
func TestIntegration(t *testing.T) {
	fstests.Run(t, &fstests.Opt{
		RemoteName: "TestIA:lesmi-zclone-test/",
		NilObject:  (*internetarchive.Object)(nil),
	})
}
