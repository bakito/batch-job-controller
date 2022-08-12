package test

import (
	"os"
	"path/filepath"
)

func TempDir(id string) (string, error) {
	dir, err := os.MkdirTemp("", "go-test-")
	if err != nil {
		return "", err
	}
	return dir, os.MkdirAll(filepath.Join(dir, id), os.ModePerm)
}
