package utils

import (
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
