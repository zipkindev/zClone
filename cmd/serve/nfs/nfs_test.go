//go:build unix

// The serving is tested in cmd/nfsmount - here we test anything else
package nfs

import (
	"testing"

	_ "zclone/backend/local"
	"zclone/cmd/serve/servetest"
	"zclone/fs/rc"
)

func TestRc(t *testing.T) {
	servetest.TestRc(t, rc.Params{
		"type":           "nfs",
		"vfs_cache_mode": "off",
	})
}
