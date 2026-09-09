package internxt_test

import (
	"testing"

	"zclone/fs"
	"zclone/fstest/fstests"
)

// TestIntegration runs integration tests against the remote
func TestIntegration(t *testing.T) {
	fstests.Run(t, &fstests.Opt{
		RemoteName: "TestInternxt:",
		ChunkedUpload: fstests.ChunkedUploadConfig{
			MinChunkSize:       100 * fs.Mebi,
			NeedMultipleChunks: true,
		},
	})
}
