package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

func allocatedBytes(name string) (uint64, error) {
	pointer, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}
	var high uint32
	low, _, callErr := syscall.NewLazyDLL("kernel32.dll").NewProc("GetCompressedFileSizeW").Call(uintptr(unsafe.Pointer(pointer)), uintptr(unsafe.Pointer(&high)))
	if uint32(low) == 0xffffffff && callErr != syscall.Errno(0) {
		return 0, fmt.Errorf("allocated disk query: %w", callErr)
	}
	return uint64(high)<<32 | uint64(uint32(low)), nil
}
