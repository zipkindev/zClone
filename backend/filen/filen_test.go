package filen

import (
	"testing"

	"zclone/fstest/fstests"
)

func TestIntegration(t *testing.T) {
	fstests.Run(t, &fstests.Opt{
		RemoteName: "TestFilen:",
		NilObject:  (*Object)(nil),
	})
}
