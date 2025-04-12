package utils

import (
	"os"
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
