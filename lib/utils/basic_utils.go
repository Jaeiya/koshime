package utils

import (
	"strconv"
)

func IsNumber(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}
