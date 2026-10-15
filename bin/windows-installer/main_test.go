package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T, executable []byte, extra string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	z := zip.NewWriter(&buffer)
	files := map[string][]byte{"zclone.exe": executable, "COPYING": []byte("license"), "NOTICE": []byte("attribution"), "build-info.json": []byte(`{"os":"windows","arch":"amd64","compression":"none"}`), "build/zclone.sbom.cdx.json": []byte(`{}`)}
	if extra != "" {
		files[extra] = []byte("unsafe")
	}
	for name, data := range files {
		f, err := z.Create("release/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestInstallPreservesActiveFileOnFailure(t *testing.T) {
	for _, failure := range []string{"compress", "modified", "traversal", "duplicate"} {
		t.Run(failure, func(t *testing.T) {
			root := t.TempDir()
			target := filepath.Join(root, "installed")
			os.Mkdir(target, 0700)
			active := filepath.Join(target, "zclone.exe")
			os.WriteFile(active, []byte("old"), 0600)
			extra := ""
			if failure == "traversal" {
				extra = "../escape"
			}
			if failure == "duplicate" {
				extra = "ZCLONE.EXE"
			}
			archive := filepath.Join(root, "release.zip")
			os.WriteFile(archive, fixture(t, []byte("new"), extra), 0600)
			compressor := func(binary, algorithm string) error {
				if failure == "compress" {
					return errors.New("unavailable filesystem")
				}
				if failure == "modified" {
					return os.WriteFile(binary, []byte("changed"), 0600)
				}
				return nil
			}
			if _, err := install(archive, target, "XPRESS4K", compressor); err == nil {
				t.Fatal("accepted failed install")
			}
			data, _ := os.ReadFile(active)
			if string(data) != "old" {
				t.Fatal("active executable changed")
			}
			stages, _ := filepath.Glob(filepath.Join(target, ".zclone-stage-*"))
			if len(stages) != 0 {
				t.Fatal("staging files left behind")
			}
		})
	}
}

func TestInstallAndUpdate(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "installed")
	os.Mkdir(target, 0700)
	config := filepath.Join(target, "rclone.conf")
	os.WriteFile(config, []byte("configuration sentinel"), 0600)
	calls := 0
	for _, content := range []string{"old", "new", "new"} {
		archive := filepath.Join(root, "release.zip")
		os.WriteFile(archive, fixture(t, []byte(content), ""), 0600)
		if _, err := install(archive, target, "XPRESS4K", func(binary, algorithm string) error { calls++; return nil }); err != nil {
			t.Fatal(err)
		}
		got, _ := os.ReadFile(filepath.Join(target, "zclone.exe"))
		if string(got) != content {
			t.Fatal("wrong installed content")
		}
	}
	if calls != 3 {
		t.Fatal("compression not applied after each install")
	}
	data, _ := os.ReadFile(config)
	if string(data) != "configuration sentinel" {
		t.Fatal("configuration changed")
	}
}

func TestDownloadAndVerify(t *testing.T) {
	for _, mode := range []string{"valid", "checksum", "http", "truncated", "missing", "draft", "tag"} {
		t.Run(mode, func(t *testing.T) {
			payload := fixture(t, []byte("downloaded"), "")
			digest := sha256.Sum256(payload)
			checksum := hex.EncodeToString(digest[:])
			if mode == "checksum" {
				checksum = strings.Repeat("0", 64)
			}
			var server *httptest.Server
			server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/latest", "/tags/v1":
					tag := "v1"
					if mode == "tag" {
						tag = "v2"
					}
					assets := []map[string]string{{"name": "zclone-" + tag + "-windows-amd64.zip", "browser_download_url": server.URL + "/archive"}, {"name": "SHA256SUMS", "browser_download_url": server.URL + "/checksums"}}
					if mode == "missing" {
						assets = nil
					}
					json.NewEncoder(w).Encode(map[string]any{"tag_name": tag, "draft": mode == "draft", "assets": assets})
				case "/checksums":
					fmt.Fprintf(w, "%s  zclone-v1-windows-amd64.zip\n", checksum)
				case "/archive":
					if mode == "http" {
						http.Error(w, "unavailable", 503)
						return
					}
					if mode == "truncated" {
						payload = payload[:len(payload)/2]
					}
					w.Write(payload)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			root := t.TempDir()
			file, err := downloadRelease(server.Client(), server.URL+"/", "v1", root, false)
			if mode == "valid" {
				if err != nil {
					t.Fatal(err)
				}
				data, _ := os.ReadFile(file)
				if !bytes.Equal(data, payload) {
					t.Fatal("download changed")
				}
			} else if err == nil {
				t.Fatal("accepted invalid download")
			}
		})
	}
}

func TestNativeStreamedInstall(t *testing.T) {
	archive := os.Getenv("ZCLONE_INSTALLER_QA_ARCHIVE")
	if archive == "" {
		t.Skip("requires a disposable Windows release archive")
	}
	content, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(content)
	var server *httptest.Server
	server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest":
			json.NewEncoder(w).Encode(map[string]any{"tag_name": "v0.1.0", "assets": []map[string]string{{"name": "zclone-v0.1.0-windows-amd64.zip", "browser_download_url": server.URL + "/archive"}, {"name": "SHA256SUMS", "browser_download_url": server.URL + "/checksums"}}})
		case "/checksums":
			fmt.Fprintf(w, "%x  zclone-v0.1.0-windows-amd64.zip\n", digest)
		case "/archive":
			reader, err := os.Open(archive)
			if err != nil {
				t.Error(err)
				return
			}
			defer reader.Close()
			io.Copy(w, reader)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	download, err := downloadRelease(server.Client(), server.URL+"/", "", t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "installed")
	hash, err := install(download, root, "XPRESS4K", nativeCompress)
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(root, "zclone.exe")
	actual, err := fileHash(exe)
	if err != nil || actual != hash {
		t.Fatal("installed hash mismatch", err)
	}
	allocation, err := allocatedBytes(exe)
	if err != nil {
		t.Fatal(err)
	}
	stat, err := os.Stat(exe)
	if err != nil || allocation >= uint64(stat.Size()) {
		t.Fatal("installed executable not compressed", err)
	}
}

func TestLatestPublishedSelection(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" && r.URL.Query().Get("per_page") == "100" {
			io.WriteString(w, `[{"tag_name":"draft","draft":true},{"tag_name":"v0.1.0","prerelease":true,"assets":[]}]`)
			return
		}
		t.Errorf("unexpected discovery endpoint: %s", r.URL)
		http.NotFound(w, r)
	}))
	defer server.Close()
	_, err := downloadRelease(server.Client(), server.URL+"/", "", t.TempDir(), true)
	if err == nil || !strings.Contains(err.Error(), "release lacks ordinary") {
		t.Fatal("latest published selection did not reach prerelease assets", err)
	}
}

func TestPublishedReleaseDownload(t *testing.T) {
	if os.Getenv("ZCLONE_INSTALLER_LIVE_QA") != "1" {
		t.Skip("explicit public release download QA only")
	}
	archive, err := downloadRelease(httpClient(), "https://api.github.com/repos/zipkindev/zClone/releases/", "", t.TempDir(), true)
	if err != nil {
		t.Fatal(err)
	}
	if err = extract(archive, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Log("Published release discovery, streamed download, checksum, and extraction passed")
}

func TestDestinationPreferences(t *testing.T) {
	root := t.TempDir()
	settings := filepath.Join(root, "preferences", "settings.json")
	fallback := filepath.Join(root, "default")
	got, err := rememberedDirectory(settings, fallback)
	if err != nil || got != fallback {
		t.Fatal("default destination", got, err)
	}
	for _, name := range []string{"custom location", "new destination"} {
		destination := filepath.Join(root, name)
		if err = rememberDirectory(settings, destination); err != nil {
			t.Fatal(err)
		}
		got, err = rememberedDirectory(settings, fallback)
		if err != nil || got != destination {
			t.Fatal("remembered destination", got, err)
		}
	}
	if err = os.WriteFile(settings, []byte(`{"install_directory":"relative"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = rememberedDirectory(settings, fallback); err == nil {
		t.Fatal("accepted malformed saved destination")
	}
}

func TestDownloadProgress(t *testing.T) {
	var statuses []string
	var percentages []int
	writer := downloadProgress{total: 8, progress: func(status string, percent int) {
		statuses = append(statuses, status)
		percentages = append(percentages, percent)
	}}
	writer.Write([]byte("1234"))
	writer.Write([]byte("5678"))
	if len(statuses) != 2 || !strings.Contains(statuses[0], "Downloading") || percentages[0] != 35 || percentages[1] != 65 {
		t.Fatal("incorrect download progress", statuses, percentages)
	}
	writer = downloadProgress{progress: func(status string, percent int) {
		if percent != -1 {
			t.Fatal("unknown download size must be indeterminate")
		}
	}}
	writer.Write([]byte("stream"))
}
