// The experimental updater stages an ordinary release before native compression.
package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

func fileHash(name string) (string, error) {
	f, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func extract(archive, destination string) error {
	z, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer z.Close()
	root := ""
	seen := map[string]bool{}
	for _, f := range z.File {
		parts := strings.SplitN(f.Name, "/", 2)
		if len(parts) != 2 || parts[0] == "" {
			return errors.New("payload requires one release directory")
		}
		if root == "" {
			root = parts[0]
		}
		relative := parts[1]
		if parts[0] != root || strings.ContainsAny(f.Name, "\\:") || strings.HasPrefix(relative, "/") || path.Clean(relative) != relative || relative == "." || relative == ".." || strings.HasPrefix(relative, "../") || f.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe payload path %q", f.Name)
		}
		key := strings.ToLower(relative)
		if seen[key] {
			return fmt.Errorf("duplicate payload path %q", f.Name)
		}
		seen[key] = true
		target := filepath.Join(destination, filepath.FromSlash(relative))
		if f.FileInfo().IsDir() {
			return errors.New("payload must contain files only")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		if err := extractFile(f, target); err != nil {
			return err
		}
	}
	for _, name := range []string{"zclone.exe", "COPYING", "NOTICE", "build-info.json", "build/zclone.sbom.cdx.json"} {
		if !seen[strings.ToLower(name)] {
			return fmt.Errorf("missing payload file %s", name)
		}
	}
	var info struct{ OS, Arch, Compression string }
	data, err := os.ReadFile(filepath.Join(destination, "build-info.json"))
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, &info); err != nil {
		return err
	}
	if info.OS != "windows" || info.Arch != "amd64" || (info.Compression != "none" && info.Compression != "") {
		return errors.New("payload is not an ordinary Windows amd64 release")
	}
	return nil
}

func extractFile(f *zip.File, target string) error {
	input, err := f.Open()
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	return errors.Join(copyErr, closeErr)
}

func nativeCompress(binary, algorithm string) error {
	command := exec.Command(filepath.Join(os.Getenv("SystemRoot"), "System32", "compact.exe"), "/C", "/F", "/EXE:"+algorithm, binary)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("native compression requires a supported destination filesystem: %w: %s", err, output)
	}
	allocated, err := allocatedBytes(binary)
	if err != nil {
		return err
	}
	info, err := os.Stat(binary)
	if err != nil {
		return err
	}
	if allocated >= uint64(info.Size()) {
		return errors.New("native compression did not reduce allocated disk usage")
	}
	return nil
}

func install(archive, directory, algorithm string, compress func(string, string) error) (string, error) {
	if algorithm != "XPRESS4K" && algorithm != "XPRESS8K" && algorithm != "LZX" {
		return "", errors.New("unsupported compression algorithm")
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return "", err
	}
	stage, err := os.MkdirTemp(directory, ".zclone-stage-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	if err = extract(archive, stage); err != nil {
		return "", err
	}
	binary := filepath.Join(stage, "zclone.exe")
	before, err := fileHash(binary)
	if err != nil {
		return "", err
	}
	if err = compress(binary, algorithm); err != nil {
		return "", err
	}
	after, err := fileHash(binary)
	if err != nil {
		return "", err
	}
	if before != after {
		return "", errors.New("compression changed executable content")
	}
	// Metadata is versioned so a failed executable replacement preserves its matching records.
	metadata := filepath.Join(directory, ".zclone-releases", before)
	if err = os.MkdirAll(metadata, 0700); err != nil {
		return "", err
	}
	for _, name := range []string{"COPYING", "NOTICE", "README.md", "DEPENDENCY_MANIFEST.sha256", "build-info.json", "build", "bin", "docs", "RELEASE.md", "compress-windows.ps1"} {
		source := filepath.Join(stage, name)
		if _, err = os.Stat(source); os.IsNotExist(err) {
			continue
		}
		target := filepath.Join(metadata, name)
		if _, err = os.Stat(target); err == nil {
			continue
		}
		if err = os.Rename(source, target); err != nil {
			return "", err
		}
	}
	// Windows rejects replacement of a running image; the existing file stays in place.
	if err = os.Rename(binary, filepath.Join(directory, "zclone.exe")); err != nil {
		return "", fmt.Errorf("replacement failed; close running zclone processes and retry: %w", err)
	}
	return before, nil
}

func runUpdater() error {
	defaultDirectory := filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "zclone")
	directory := flag.String("install-dir", "", "installation directory; default is remembered destination or "+defaultDirectory)
	algorithm := flag.String("algorithm", "XPRESS4K", "native compression: XPRESS4K, XPRESS8K, or LZX")
	if runtime.GOOS != "windows" {
		return errors.New("this experimental updater requires Windows")
	}
	archive := flag.String("archive", "", "offline release ZIP (requires --sha256)")
	digest := flag.String("sha256", "", "expected SHA-256 for an offline release ZIP")
	tag := flag.String("release", "", "specific release tag; default is latest stable release")
	stable := flag.Bool("stable", false, "select GitHub latest stable release (default)")
	prerelease := flag.Bool("prerelease", false, "include published prereleases when selecting the newest release")
	flag.Parse()
	if *stable && *prerelease {
		return errors.New("--stable and --prerelease cannot be combined")
	}
	if flag.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	settings := filepath.Join(os.Getenv("LOCALAPPDATA"), "zclone-updater", "settings.json")
	if *directory == "" {
		saved, err := rememberedDirectory(settings, defaultDirectory)
		if err != nil {
			return fmt.Errorf("read updater destination: %w", err)
		}
		*directory = saved
	}
	absolute, err := filepath.Abs(*directory)
	if err != nil {
		return err
	}
	*directory = absolute
	temporary, err := os.MkdirTemp("", "zclone-download-")
	if err == nil {
		defer os.RemoveAll(temporary)
	}
	source := *archive
	if err == nil && source == "" {
		source, err = downloadRelease(httpClient(), "https://api.github.com/repos/zipkindev/zClone/releases/", *tag, temporary, *prerelease)
	}
	if err == nil && *archive != "" {
		var actual string
		actual, err = fileHash(source)
		if err == nil && (len(*digest) != 64 || !strings.EqualFold(actual, *digest)) {
			err = errors.New("offline archive SHA-256 mismatch")
		}
	}
	if err == nil {
		var digest string
		digest, err = install(source, *directory, *algorithm, nativeCompress)
		if err == nil {
			if saveErr := rememberDirectory(settings, *directory); saveErr != nil {
				fmt.Fprintln(os.Stderr, "Installed successfully, but could not remember destination:", saveErr)
			}
			return json.NewEncoder(os.Stdout).Encode(map[string]string{"path": filepath.Join(*directory, "zclone.exe"), "sha256": digest, "algorithm": *algorithm, "status": "installed"})
		}
	}
	return err
}

func main() {
	if err := runUpdater(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
