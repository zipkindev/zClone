// Package cmdtest creates a testable interface to zclone main
//
// The interface is used to perform end-to-end test of
// commands, flags, environment variables etc.
package cmdtest

// The rest of this file is a 1:1 copy from zclone.go

import (
	_ "zclone/backend/all" // import all backends
	"zclone/cmd"
	_ "zclone/cmd/all"    // import all commands
	_ "zclone/lib/plugin" // import plugins
)

func main() {
	cmd.Main()
}
