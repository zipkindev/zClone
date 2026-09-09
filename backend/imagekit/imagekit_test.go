package imagekit

import (
	"testing"

	"zclone/fstest"
	"zclone/fstest/fstests"
)

func TestIntegration(t *testing.T) {
	debug := true
	fstest.Verbose = &debug
	fstests.Run(t, &fstests.Opt{
		RemoteName:      "TestImageKit:",
		NilObject:       (*Object)(nil),
		SkipFsCheckWrap: true,
	})
}
