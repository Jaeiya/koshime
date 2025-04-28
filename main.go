package main

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/Jaeiya/koshime/lib/utils"
)

var (
	errUserAborted = fmt.Errorf("user aborted initialization")
	dbFilePath     = filepath.Join("./", "koshime.db")
)

func main() {
	fmt.Println("")
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

func InitDatabase(path string) (*database.Database, error) {
	var db *database.Database
	var err error
	if !utils.FileExists(path) {
		data, isAborted := ui.NewUser()
		if isAborted {
			return db, errUserAborted
		}
		db, err = database.NewDatabase(&data)
		if err != nil {
			return db, err
		}
		return db, nil
	}

	db, err = database.NewDatabase(nil)
	err = db.Load()
	if err != nil {
		return db, err
	}
	return db, nil
}
