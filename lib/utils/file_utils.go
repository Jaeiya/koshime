package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetWorkingDir gets the working directory without returning
// an error.
//
// 🔴 Panics on error. We always expect file
// operations to succeed.
func GetWorkingDir() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return wd
}

// GetExecutableDirPath returns the directory where this
// program is executed from.
//
// 🔴 Panics on error. We always expect file
// operations to succeed.
func GetExecutableDirPath() string {
	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}
	realPath, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		panic(err)
	}
	return filepath.Dir(realPath)
}

// FileExists returns true if the specified file
// path has been found.
//
// 🔴 Panics on error. We always expect file
// operations to succeed.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		panic(err)
	}
	return !info.IsDir()
}

type FilenameIterator interface {
	Next() (fileName string, ok bool)
}

type FilenameStream struct {
	entries []os.DirEntry
	index   int
}

// NewFilenameStream reads the specified directory and returns a
// struct which allows you to walk through each file name.
//
// 🟡 Directory names are ignored
func NewFilenameStream(dir string) (*FilenameStream, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return &FilenameStream{}, fmt.Errorf("could not read dir: %w", err)
	}
	return &FilenameStream{entries, 0}, nil
}

// Next returns the next file name and its status. If there
// are no more file names, the status is false.
func (fs *FilenameStream) Next() (string, bool) {
	if fs.index == len(fs.entries) {
		return "", false
	}
	fileName := fs.entries[fs.index].Name()
	fs.index++
	return fileName, true
}
