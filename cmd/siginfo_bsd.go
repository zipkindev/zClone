//go:build darwin || freebsd || netbsd || dragonfly || openbsd

package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"zclone/fs"
	"zclone/fs/accounting"
)

// SigInfoHandler creates SigInfo handler
func SigInfoHandler() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINFO)
	go func() {
		for range signals {
			fs.Printf(nil, "%v\n", accounting.GlobalStats())
		}
	}()
}
