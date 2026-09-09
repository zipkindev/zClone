// Test DOI filesystem interface
package doi

import (
	"testing"

	"zclone/fstest/fstests"
)

// TestIntegration runs integration tests against the remote
func TestIntegration(t *testing.T) {
	fstests.Run(t, &fstests.Opt{
		RemoteName: "TestDoi:",
		NilObject:  (*Object)(nil),
	})
}
