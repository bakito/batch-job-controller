package test

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

func TempDir(id string) (string, error) {
	dir, err := ioutil.TempDir("", "go-test-")
	if err != nil {
		return "", err
	}
	return dir, os.MkdirAll(filepath.Join(dir, id), os.ModePerm)
}
