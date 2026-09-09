// Package gomobile exports shims for gomobile use
package gomobile

import (
	"zclone/libzclone/libzclone"

	_ "zclone/backend/all" // import all backends
	_ "zclone/lib/plugin"  // import plugins

	_ "golang.org/x/mobile/event/key" // make go.mod add this as a dependency
)

// ZcloneInitialize initializes zclone as a library
func ZcloneInitialize() {
	libzclone.Initialize()
}

// ZcloneFinalize finalizes the library
func ZcloneFinalize() {
	libzclone.Finalize()
}

// ZcloneRPCResult is returned from ZcloneRPC
//
//	Output will be returned as a serialized JSON object
//	Status is a HTTP status return (200=OK anything else fail)
type ZcloneRPCResult struct {
	Output string
	Status int
}

// ZcloneRPC has an interface optimised for gomobile, in particular
// the function signature is valid under gobind rules.
//
// https://pkg.go.dev/golang.org/x/mobile/cmd/gobind#hdr-Type_restrictions
func ZcloneRPC(method string, input string) (result *ZcloneRPCResult) { //nolint:deadcode
	output, status := libzclone.RPC(method, input)
	return &ZcloneRPCResult{
		Output: output,
		Status: status,
	}
}
