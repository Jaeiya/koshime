package utils

import (
	"fmt"
	"strconv"
)

type ByteMap struct {
	size     int64
	notation string
}

var byteTable = []ByteMap{
	{1 << (10 * 5), "PiB"},
	{1 << (10 * 4), "TiB"},
	{1 << (10 * 3), "GiB"},
	{1 << (10 * 2), "MiB"},
	{1 << (10 * 1), "KiB"},
	{0, "Bytes"},
}

func FormatBytes(byteSize int64) string {
	for _, entry := range byteTable {
		if byteSize >= entry.size {
			return fmt.Sprintf("%.2f %s", float64(byteSize)/float64(entry.size), entry.notation)
		}
	}
	return "0 bytes"
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
