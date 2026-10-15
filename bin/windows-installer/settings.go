package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type preferences struct {
	InstallDirectory string `json:"install_directory"`
}

func rememberedDirectory(settings, fallback string) (string, error) {
	data, err := os.ReadFile(settings)
	if errors.Is(err, os.ErrNotExist) {
		return fallback, nil
	}
	if err != nil {
		return "", err
	}
	var saved preferences
	if err = json.Unmarshal(data, &saved); err != nil {
		return "", err
	}
	if !filepath.IsAbs(saved.InstallDirectory) {
		return "", errors.New("saved installation directory must be absolute; use --install-dir to choose a destination")
	}
	return saved.InstallDirectory, nil
}

func rememberDirectory(settings, directory string) error {
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(preferences{InstallDirectory: absolute}, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(settings), 0700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(settings), "settings-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	_, writeErr := temporary.Write(append(data, '\n'))
	closeErr := temporary.Close()
	if err = errors.Join(writeErr, closeErr); err != nil {
		return err
	}
	return os.Rename(temporary.Name(), settings)
}
