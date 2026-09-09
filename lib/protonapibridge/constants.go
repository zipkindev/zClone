package proton_api_bridge

var (
	LIB_VERSION = "1.0.0"

	UPLOAD_BLOCK_SIZE       = 4 * 1024 * 1024 // 4 MB
	UPLOAD_BATCH_BLOCK_SIZE = 8
	/*
		the upstream implementation

		The idea is that Zclone performs buffering / pre-fetching on it's own so
		we don't need to be doing this on our end.

		If you are not using Zclone and instead is directly basing your work on this
		library, then maybe you can increase this value to let the library does
		the buffering work for you!
	*/
	DOWNLOAD_BATCH_BLOCK_SIZE = 1
)
