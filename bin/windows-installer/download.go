package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// httpClient bounds both slow downloads and redirects while retaining proxy support.
func httpClient() *http.Client {
	return &http.Client{Timeout: 15 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 || req.URL.Scheme != "https" {
			return errors.New("unsafe release redirect")
		}
		return nil
	}}
}

func get(client *http.Client, address string) (*http.Response, error) {
	request, err := http.NewRequest(http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "zClone-Updater")
	request.Header.Set("Accept", "application/vnd.github+json")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return nil, fmt.Errorf("release request returned HTTP %d", response.StatusCode)
	}
	return response, nil
}

func downloadRelease(client *http.Client, api, tag, directory string, includePrerelease bool) (string, error) {
	return downloadReleaseProgress(client, api, tag, directory, includePrerelease, nil)
}

func downloadReleaseProgress(client *http.Client, api, tag, directory string, includePrerelease bool, progress progressFunc) (string, error) {
	progress.send("Finding the latest release…", 0)
	endpoint := api + "latest"
	if includePrerelease && tag == "" {
		endpoint = strings.TrimSuffix(api, "/") + "?per_page=100"
	}
	if tag != "" {
		endpoint = api + "tags/" + url.PathEscape(tag)
	}
	response, err := get(client, endpoint)
	if err != nil {
		return "", err
	}
	type releaseInfo struct {
		Tag    string `json:"tag_name"`
		Draft  bool   `json:"draft"`
		Assets []struct {
			Name string
			URL  string `json:"browser_download_url"`
			Size int64
		} `json:"assets"`
	}
	var release releaseInfo
	decoder := json.NewDecoder(io.LimitReader(response.Body, 4<<20))
	if includePrerelease && tag == "" {
		var releases []releaseInfo
		err = decoder.Decode(&releases)
		for _, candidate := range releases {
			if !candidate.Draft {
				release = candidate
				break
			}
		}
	} else {
		err = decoder.Decode(&release)
	}
	response.Body.Close()
	if err != nil {
		return "", err
	}
	if release.Draft || release.Tag == "" || (tag != "" && release.Tag != tag) {
		return "", errors.New("release identity mismatch or unpublished release")
	}
	name := "zclone-" + release.Tag + "-windows-amd64.zip"
	assets := map[string]string{}
	var archiveSize int64
	for _, asset := range release.Assets {
		if asset.Name == name || asset.Name == "SHA256SUMS" {
			if assets[asset.Name] != "" {
				return "", errors.New("ambiguous release asset")
			}
			assets[asset.Name] = asset.URL
			if asset.Name == name {
				archiveSize = asset.Size
			}
		}
	}
	if assets[name] == "" || assets["SHA256SUMS"] == "" {
		return "", errors.New("release lacks ordinary Windows amd64 archive or SHA256SUMS")
	}
	// Production discovery is fixed to this repository; test servers use a separate client/API.
	if strings.HasPrefix(api, "https://api.github.com/") {
		for _, address := range assets {
			u, err := url.Parse(address)
			if err != nil || u.Scheme != "https" || u.Host != "github.com" || !strings.HasPrefix(u.Path, "/zipkindev/zClone/releases/download/") {
				return "", errors.New("unexpected release asset origin")
			}
		}
	}
	response, err = get(client, assets["SHA256SUMS"])
	if err != nil {
		return "", err
	}
	checksums, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	response.Body.Close()
	if err != nil {
		return "", err
	}
	expected := ""
	for _, line := range strings.Split(string(checksums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == name {
			if expected != "" {
				return "", errors.New("ambiguous archive checksum")
			}
			expected = strings.ToLower(fields[0])
		}
	}
	if decoded, err := hex.DecodeString(expected); err != nil || len(decoded) != sha256.Size {
		return "", errors.New("missing or invalid archive checksum")
	}
	response, err = get(client, assets[name])
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	destination := filepath.Join(directory, "release.zip")
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	hasher := sha256.New()
	// Stream to disk with bounded memory and reject unexpectedly large responses.
	count, copyErr := io.Copy(io.MultiWriter(output, hasher, &downloadProgress{total: archiveSize, progress: progress}), io.LimitReader(response.Body, (512<<20)+1))
	closeErr := output.Close()
	if err = errors.Join(copyErr, closeErr); err != nil {
		return "", err
	}
	progress.send("Verifying download…", 67)
	if count > 512<<20 || hex.EncodeToString(hasher.Sum(nil)) != expected {
		return "", errors.New("downloaded archive SHA-256 mismatch or size limit exceeded")
	}
	return destination, nil
}
