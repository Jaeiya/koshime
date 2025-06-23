package utils

import (
	"fmt"
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

func CalcRating(r string) string {
	if r == "" {
		return ""
	}

	rawRating, err := strconv.ParseFloat(r, 64)
	if err != nil {
		panic(fmt.Errorf("could not calc avg rating: %w", err))
	}

	return fmt.Sprintf("%.2f", rawRating/10)
}
