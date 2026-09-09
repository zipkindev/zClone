// Sync files and directories to and from local and remote object stores
//
// Nick Craig-Wood <nick@craig-wood.com>
package main

import (
	_ "zclone/backend/all" // import all backends
	"zclone/cmd"
	_ "zclone/cmd/all"    // import all commands
	_ "zclone/lib/plugin" // import plugins
)

func main() {
	cmd.Main()
}
