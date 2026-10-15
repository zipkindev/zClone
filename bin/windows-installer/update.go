package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type progressFunc func(string, int)

func (p progressFunc) send(status string, percent int) {
	if p != nil {
		p(status, percent)
	}
}

type downloadProgress struct {
	done, total int64
	last        time.Time
	progress    progressFunc
}

func (d *downloadProgress) Write(data []byte) (int, error) {
	d.done += int64(len(data))
	if time.Since(d.last) >= 100*time.Millisecond || d.done == d.total {
		d.last = time.Now()
		if d.total > 0 {
			percent := min(65, 5+int(d.done*60/d.total))
			d.progress.send(fmt.Sprintf("Downloading %.1f of %.1f MB…", float64(d.done)/1e6, float64(d.total)/1e6), percent)
		} else {
			d.progress.send(fmt.Sprintf("Downloading %.1f MB…", float64(d.done)/1e6), -1)
		}
	}
	return len(data), nil
}

type updateOptions struct {
	Directory, Algorithm, Archive, Digest, Tag string
	Prerelease                                 bool
}

type updateResult struct {
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	Algorithm string `json:"algorithm"`
	Status    string `json:"status"`
	Warning   string `json:"warning,omitempty"`
}

func performUpdate(options updateOptions, progress progressFunc) (updateResult, error) {
	var result updateResult
	if strings.TrimSpace(options.Directory) == "" {
		return result, errors.New("choose an installation folder")
	}
	absolute, err := filepath.Abs(options.Directory)
	if err != nil {
		return result, err
	}
	temporary, err := os.MkdirTemp("", "zclone-download-")
	if err != nil {
		return result, err
	}
	defer os.RemoveAll(temporary)
	source := options.Archive
	if source == "" {
		source, err = downloadReleaseProgress(httpClient(), "https://api.github.com/repos/zipkindev/zClone/releases/", options.Tag, temporary, options.Prerelease, progress)
	} else {
		progress.send("Verifying download…", 67)
		var actual string
		actual, err = fileHash(source)
		if err == nil && (len(options.Digest) != 64 || !strings.EqualFold(actual, options.Digest)) {
			err = errors.New("offline archive SHA-256 mismatch")
		}
	}
	if err != nil {
		return result, err
	}
	digest, err := installProgress(source, absolute, options.Algorithm, nativeCompress, progress)
	if err != nil {
		return result, err
	}
	result = updateResult{Path: filepath.Join(absolute, "zclone.exe"), SHA256: digest, Algorithm: options.Algorithm, Status: "installed"}
	settings := filepath.Join(os.Getenv("LOCALAPPDATA"), "zclone-updater", "settings.json")
	if err = rememberDirectory(settings, absolute); err != nil {
		result.Warning = "Installed successfully, but could not remember the folder: " + err.Error()
	}
	progress.send("zClone is ready to use.", 100)
	return result, nil
}
