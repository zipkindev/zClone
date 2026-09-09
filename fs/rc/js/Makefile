build:
	GOARCH=wasm GOOS=js go build -o zclone.wasm

serve:	build
	go run serve.go
