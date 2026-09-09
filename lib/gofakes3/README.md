![Logo](/GoFakeS3.png)

[![Build Status](https://zclone/lib/gofakes3/workflows/build/badge.svg)](https://zclone/lib/gofakes3/actions?query=workflow%3Abuild)
[![Go Report Card](https://goreportcard.com/badge/zclone/lib/gofakes3)](https://goreportcard.com/report/zclone/lib/gofakes3)
[![GoDoc](https://pkg.go.dev/badge/zclone/lib/gofakes3.svg)](https://pkg.go.dev/zclone/lib/gofakes3)

This is a fork of [johannesboyne/gofakes3](https://github.com/johannesboyne/gofakes3)
mainly for use implementing the Zclone S3 server command.

Notable differences:

* Use modified xml library to handle more control chars
* Func `getVersioningConfiguration` will return empty when unversioned
* New func in `backend` interface: `CopyObject`
* Support authentication with AWS Signature V4 
* Interfaces changed to take context
