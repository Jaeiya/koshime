package main

import (
	"fmt"

	"github.com/jaeiya/koshime/lib"
)

func main() {
	var fs lib.Fansub
	fileName := "[Erai-raws] Wind Breaker Season 2 - 01 [1080p CR WEBRip HEVC EAC3][MultiSub][4A372"
	info, err := fs.Parse(fileName)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", info)
	return
}
