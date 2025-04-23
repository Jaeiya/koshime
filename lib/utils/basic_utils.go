package utils

import (
	"os"
	"path/filepath"
	"strconv"
)

// GetWorkingDir gets the working directory without returning
// an error.
//
// 🔴 Will panic if an error occurs. This function should
// never error, so we panic on unexpected behavior.
func GetWorkingDir() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return wd
}

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

func IsNumber(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func AbsInt(i int) int {
	if i < 0 {
		return i * -1
	}
	return i
}

func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		// Unexpected error: permissions, disk, etc...
		panic(err)
	}

	return !info.IsDir()
}
