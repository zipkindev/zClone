package cmount

// ProvidedBy returns true if the zclone build for the given OS
// provides support for lib/cgo-fuse
func ProvidedBy(osName string) bool {
	return osName == "windows" || osName == "darwin"
}
