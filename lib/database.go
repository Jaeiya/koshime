package lib

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jaeiya/koshime/lib/utils"
	"github.com/shamaton/msgpack/v2"
)

var dbFileName = "koshime.db"

type DBProfile struct {
	ID              string
	SecondsWatched  int
	CompletedSeries int
	Username        string
	AccessToken     string
	RefreshToken    string
	TokenExpiration int
}

type DBLibEntry struct {
	// Anime ID
	ID string
	// Anime User-library ID - Allows looking up User-specific Anime data
	LibID     string
	JPN_Title string
	ENG_Title string
	Synonyms  []string
	Episodes  int
	Progress  int
	Slug      string
}

type DBData struct {
	Profile DBProfile
	Library []DBLibEntry
}

type Database struct {
	isLoaded bool
	data     DBData
}

func (Database) NewDatabase(data DBData) (*Database, error) {
	db := &Database{false, data}
	db.isLoaded = true
	err := db.Save()
	if err != nil {
		return &Database{}, err
	}
	return db, nil
}

func (db *Database) Load() error {
	if db.isLoaded {
		return nil
	}
	file, err := os.ReadFile(filepath.Join(utils.GetWorkingDir(), dbFileName))
	if err != nil {
		return err
	}
	msgpack.Unmarshal(file, &db.data)
	db.isLoaded = true
	return nil
}

func (db Database) Save() error {
	bytes, err := msgpack.Marshal(db.data)
	if err != nil {
		return err
	}
	os.WriteFile(dbFileName, bytes, 0o644)
	return nil
}

func (db Database) GetData() (*DBData, error) {
	if !db.isLoaded {
		return &DBData{}, fmt.Errorf("database has not been loaded")
	}
	return &db.data, nil
}
