package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

const watchDirName = "(watched)"

type FileSys struct{}

// GetWorkingDir gets the working directory without returning
// an error.
//
// 🔴 Panics on error. We always expect standard file
// operations to succeed.
func (FileSys) GetWorkingDir() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return wd
}

// GetExecutableDirPath returns the directory where this
// program is executed from.
//
// 🔴 Panics on error. We always expect standard file
// operations to succeed.
func (FileSys) GetExecutableDirPath() string {
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
// 🔴 Panics on error. We always expect standard file
// operations to succeed.
func (FileSys) FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		panic(err)
	}
	return !info.IsDir()
}

func (fs FileSys) WatchDir() string {
	wd := fs.GetWorkingDir()
	return filepath.Join(wd, watchDirName)
}

func (fs FileSys) ReadDirFiles(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	fileNames := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fileNames = append(fileNames, entry.Name())
	}

	return fileNames, nil
}

func (FileSys) MoveFile(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

type FilenameIterator interface {
	Next() (fileName string, ok bool)
}

type FilenameStream struct {
	entries []string
	index   int
}

// NewFilenameStream reads the specified directory and returns a
// struct which allows you to walk through each file name.
//
// 🟡 Directory names are ignored
func (FileSys) NewFilenameStream(dir string) (*FilenameStream, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return &FilenameStream{}, fmt.Errorf("could not read dir: %w", err)
	}

	newEntries := make([]string, len(entries))
	for i, entry := range entries {
		if entry.IsDir() {
			continue
		}
		newEntries[i] = entry.Name()
	}

	return &FilenameStream{
		entries: newEntries,
	}, nil
}

// Next returns the next file name and its status. If there
// are no more file names, the status is false.
func (fs *FilenameStream) Next() (string, bool) {
	if fs.index == len(fs.entries) {
		return "", false
	}
	fileName := fs.entries[fs.index]
	fs.index++
	return fileName, true
}
