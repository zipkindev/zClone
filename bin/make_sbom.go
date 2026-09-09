//go:build ignore

// make_sbom writes a deterministic CycloneDX module inventory and source digest.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type module struct {
	Path      string
	Version   string
	LocalPath string
}

type property struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type component struct {
	Type       string     `json:"type"`
	Name       string     `json:"name"`
	Version    string     `json:"version,omitempty"`
	Properties []property `json:"properties,omitempty"`
}

type bom struct {
	BomFormat string `json:"bomFormat"`
	Spec      string `json:"specVersion"`
	Version   int    `json:"version"`
	Metadata  struct {
		Component component `json:"component"`
	} `json:"metadata"`
	Components []component `json:"components"`
}

func main() {
	output := flag.String("output", "build/zclone.sbom.cdx.json", "SBOM output path")
	checksums := flag.String("checksums", "build/zclone.sources.sha256", "source checksum output path")
	flag.Parse()

	modules, err := listModules()
	if err != nil {
		fatal(err)
	}
	if err = writeBOM(*output, modules); err != nil {
		fatal(err)
	}
	if err = writeChecksums(*checksums); err != nil {
		fatal(err)
	}
}

func listModules() ([]module, error) {
	data, err := os.ReadFile("vendor/modules.txt")
	if err != nil {
		return nil, fmt.Errorf("read vendor module list: %w", err)
	}
	seen := map[string]bool{}
	var modules []module
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(line, "# "))
		if len(fields) < 2 || !strings.HasPrefix(fields[1], "v") || seen[fields[0]] {
			continue
		}
		seen[fields[0]] = true
		item := module{Path: fields[0], Version: fields[1]}
		if len(fields) >= 4 && fields[2] == "=>" {
			item.LocalPath = fields[3]
		}
		modules = append(modules, item)
	}
	return modules, nil
}

func writeBOM(output string, modules []module) error {
	result := bom{BomFormat: "CycloneDX", Spec: "1.5", Version: 1}
	result.Metadata.Component = component{Type: "application", Name: "zclone"}
	for _, item := range modules {
		entry := component{Type: "library", Name: item.Path, Version: item.Version}
		if item.LocalPath != "" {
			entry.Properties = []property{{Name: "zclone:local-path", Value: item.LocalPath}}
		}
		result.Components = append(result.Components, entry)
	}
	sort.Slice(result.Components, func(i, j int) bool {
		return result.Components[i].Name < result.Components[j].Name
	})
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(output, append(data, '\n'))
}

func writeChecksums(output string) error {
	var paths []string
	for _, root := range []string{"vendor", "third_party", "lib/gofakes3", "lib/internxtadapter", "lib/protonapi", "lib/protonapibridge"} {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.Mode().IsRegular() {
				paths = append(paths, filepath.ToSlash(path))
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	for _, path := range []string{
		"go.mod",
		"go.sum",
		"cmd/gui/dist.zip",
		"cmd/gui/dist.tag",
		"cmd/gui/dist/index.html",
		"cmd/gui/dist/icon.svg",
	} {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			paths = append(paths, filepath.ToSlash(path))
		}
	}
	sort.Strings(paths)
	var lines strings.Builder
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		hash := sha256.New()
		_, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		fmt.Fprintf(&lines, "%s  %s\n", hex.EncodeToString(hash.Sum(nil)), path)
	}
	return writeFile(output, []byte(lines.String()))
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
