package utils

import (
	"strconv"
)

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

func CopySlice[T any](s []T) []T {
	copiedSlice := make([]T, len(s))
	copy(copiedSlice, s)
	return copiedSlice
}
