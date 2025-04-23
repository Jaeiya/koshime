package main

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/jaeiya/koshime/lib"
	"github.com/jaeiya/koshime/lib/ui"
	"github.com/jaeiya/koshime/lib/utils"
)

var infoStr = `
Title: %s
Season: %s
Episode: %s
Encoding: %s
Source: %s
Fansub: %s

`

var (
	errUserAborted = fmt.Errorf("user aborted initialization")
	dbFilePath     = filepath.Join("./", "koshime.db")
)

func main() {
	db, err := InitDatabase(dbFilePath)
	if err != nil {
		if errors.Is(err, errUserAborted) {
			return
		}
		panic(err)
	}

	fmt.Printf("%+v", db)

	// 	fmt.Println("    Date:", entry.Date.Format("01/02/2006 3:04 PM"))
}

func InitDatabase(path string) (*lib.Database, error) {
	if !utils.FileExists(path) {
		data, isAborted := ui.NewUser()
		if isAborted {
			return &lib.Database{}, errUserAborted
		}
		db, err := lib.NewDatabase(&data)
		if err != nil {
			return &lib.Database{}, err
		}
		err = db.Save()
		if err != nil {
			return &lib.Database{}, err
		}
	}

	db, err := lib.NewDatabase(nil)
	err = db.Load()
	if err != nil {
		return &lib.Database{}, err
	}
	return db, nil
}
