// Test HDFS filesystem interface

//go:build !plan9

package hdfs_test

import (
	"testing"

	"zclone/backend/hdfs"
	"zclone/fstest/fstests"
)

// TestIntegration runs integration tests against the remote
func TestIntegration(t *testing.T) {
	fstests.Run(t, &fstests.Opt{
		RemoteName: "TestHdfs:",
		NilObject:  (*hdfs.Object)(nil),
	})
}
