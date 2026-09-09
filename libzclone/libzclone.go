// Package libzclone exports shims for C library use
//
// This directory contains code to build zclone as a C library and the
// shims for accessing zclone from C.
//
// The shims are a thin wrapper over the zclone RPC.
//
// Build a shared library like this:
//
//	go build --buildmode=c-shared -o libzclone.so zclone/libzclone
//
// Build a static library like this:
//
//	go build --buildmode=c-archive -o libzclone.a zclone/libzclone
//
// Both the above commands will also generate `libzclone.h` which should
// be `#include`d in `C` programs wishing to use the library.
//
// The library will depend on `libdl` and `libpthread`.
package main

/*
#include <stdlib.h>

struct ZcloneRPCResult {
	char*	Output;
	int	Status;
};
*/
import "C"

import (
	"unsafe"

	"zclone/libzclone/libzclone"

	_ "zclone/backend/all"   // import all backends
	_ "zclone/cmd/cmount"    // import cmount
	_ "zclone/cmd/mount"     // import mount
	_ "zclone/cmd/mount2"    // import mount2
	_ "zclone/fs/operations" // import operations/* rc commands
	_ "zclone/fs/sync"       // import sync/*
	_ "zclone/lib/plugin"    // import plugins
)

// ZcloneInitialize initializes zclone as a library
//
//export ZcloneInitialize
func ZcloneInitialize() {
	libzclone.Initialize()
}

// ZcloneFinalize finalizes the library
//
//export ZcloneFinalize
func ZcloneFinalize() {
	libzclone.Finalize()
}

// ZcloneRPCResult is returned from ZcloneRPC
//
//	Output will be returned as a serialized JSON object
//	Status is a HTTP status return (200=OK anything else fail)
type ZcloneRPCResult struct { //nolint:deadcode
	Output *C.char
	Status C.int
}

// ZcloneRPC does a single RPC call. The inputs are (method, input)
// and the output is (output, status). This is an exported interface
// to the zclone API as described in //rc/
//
//	method is a string, eg "operations/list"
//	input should be a string with a serialized JSON object
//	result.Output will be returned as a string with a serialized JSON object
//	result.Status is a HTTP status return (200=OK anything else fail)
//
// All strings are UTF-8 encoded, on all platforms.
//
// Caller is responsible for freeing the memory for result.Output
// (see ZcloneFreeString), result itself is passed on the stack.
//
//export ZcloneRPC
func ZcloneRPC(method *C.char, input *C.char) (result C.struct_ZcloneRPCResult) { //nolint:golint
	output, status := libzclone.RPC(C.GoString(method), C.GoString(input))
	result.Output = C.CString(output)
	result.Status = C.int(status)
	return result
}

// ZcloneFreeString may be used to free the string returned by ZcloneRPC
//
// If the caller has access to the C standard library, the free function can
// normally be called directly instead. In some cases the caller uses a
// runtime library which is not compatible, and then this function can be
// used to release the memory with the same library that allocated it.
//
//export ZcloneFreeString
func ZcloneFreeString(str *C.char) {
	C.free(unsafe.Pointer(str))
}

// do nothing here - necessary for building into a C library
func main() {}
