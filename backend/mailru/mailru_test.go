// Test Mailru filesystem interface
package mailru_test

import (
	"testing"

	"zclone/backend/mailru"
	"zclone/fstest/fstests"
)

// TestIntegration runs integration tests against the remote
func TestIntegration(t *testing.T) {
	fstests.Run(t, &fstests.Opt{
		RemoteName:               "TestMailru:",
		NilObject:                (*mailru.Object)(nil),
		SkipBadWindowsCharacters: true,
	})
}
