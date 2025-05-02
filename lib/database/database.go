package database

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/shamaton/msgpack/v2"
)

var dbFileName = "koshime.db"

type Profile struct {
	ID              string
	SecondsWatched  int
	CompletedSeries int
	Username        string
	About           string
	Location        string
	Birthday        string
	Gender          string
	CreatedAt       string
	AccessToken     string
	RefreshToken    string
	TokenExpiration int
	// Last time the profile was retrieved
	LastUpdateSec int64
}

type LibraryEntry struct {
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

type Data struct {
	Profile Profile
	Library []LibraryEntry
}

type Database struct {
	isLoaded bool
	data     Data
}

func NewDatabase(data *Data) (*Database, error) {
	// Initialize empty database
	if data == nil {
		db := &Database{}
		return db, nil
	}

	db := &Database{true, *data}
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

// FindAnime uses the query to do a partial lookup against
// all available anime titles, including synonyms, and
// returns all matches found.
func (db Database) FindAnime(query string) ([]LibraryEntry, error) {
	query = strings.ToLower(query)
	var entries []LibraryEntry
	for _, entry := range db.data.Library {
		if hasTitleMatches(entry, query) {
			entries = append(entries, entry)
		}
	}
	if len(entries) == 0 {
		return entries, fmt.Errorf("entry not found")
	}
	return entries, nil
}

// SaveProfile overwrites the existing profile with
// the specified one.
func (db *Database) SaveProfile(p Profile) error {
	db.data.Profile = p
	return db.Save()
}

// SaveLibrary overwrites the existing library with
// the specified one.
func (db *Database) SaveLibrary(p []LibraryEntry) error {
	db.data.Library = p
	return db.Save()
}

func (db *Database) AddLibEntry(entry LibraryEntry) error {
	db.data.Library = append(db.data.Library, entry)
	return db.Save()
}

func (db *Database) RemoveLibEntry(id string) error {
	for i, entry := range db.data.Library {
		if entry.ID == id {
			db.data.Library = slices.Delete(db.data.Library, i, i+1)
		}
	}
	return db.Save()
}

func (db Database) Save() error {
	bytes, err := msgpack.Marshal(db.data)
	if err != nil {
		return err
	}
	os.WriteFile(dbFileName, bytes, 0o644)
	return nil
}

func (db Database) GetData() (*Data, error) {
	if !db.isLoaded {
		return &Data{}, fmt.Errorf("database has not been loaded")
	}
	return &db.data, nil
}

func hasTitleMatches(e LibraryEntry, q string) bool {
	if strings.Contains(strings.ToLower(e.ENG_Title), q) ||
		strings.Contains(strings.ToLower(e.JPN_Title), q) {
		return true
	}
	return slices.ContainsFunc(e.Synonyms, func(s string) bool {
		return strings.Contains(strings.ToLower(s), q)
	})
}
