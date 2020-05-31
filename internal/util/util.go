package util

import (
	"errors"
	"os"
	"path/filepath"
)

// EnsureDirectory creates the directory if it does not exist (fail if path
// exists and is not a directory)
func EnsureDirectory(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		var fileMode os.FileMode
		fileMode = os.ModeDir | 0775
		mdErr := os.MkdirAll(path, fileMode)
		return mdErr
	}
	if err != nil {
		return err
	}
	file, _ := os.Stat(path)
	if !file.IsDir() {
		return errors.New(path + " is not a directory")
	}
	return nil
}

// CreateTmpDir makes a directory named `dir` in the system temporary directory
// and returns the full path to the created directory
func CreateTmpDir(dir string) (string, error) {
	tmpDir := os.TempDir()
	dataDir := filepath.Join(tmpDir, dir)
	err := EnsureDirectory(dataDir)
	return dataDir, err
}

// RemoveTmpDir removes a directory (and everything underneath it)
func RemoveTmpDir(dir string) error {
	return os.RemoveAll(dir)
}
