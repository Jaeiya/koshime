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

type LibraryIndex int

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
	// Initialize empty database when it needs
	// to be loaded from file
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

// Load loads an existing database. In order to initialize
// the database properly, you can either pass the data
// directly to the constructor or use this function
// to load it from file.
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

// FindLibAnimeIndex uses the query to do a partial lookup against
// all available anime titles, including synonyms, and returns
// the library index of all matches found.
func (db Database) FindLibAnimeIndex(query string) ([]LibraryIndex, error) {
	query = strings.ToLower(query)
	var indexes []LibraryIndex
	for i, entry := range db.data.Library {
		if hasTitleMatches(entry, query) {
			indexes = append(indexes, LibraryIndex(i))
		}
	}
	if len(indexes) == 0 {
		return indexes, fmt.Errorf("entry not found")
	}
	return indexes, nil
}

// GetAnime will retrieve library anime using the specified
// library indexes.
func (db Database) GetAnime(libIndexes ...LibraryIndex) ([]LibraryEntry, error) {
	if len(libIndexes) == 0 {
		return []LibraryEntry{}, fmt.Errorf("missing indexes to lookup")
	}
	entries := make([]LibraryEntry, len(libIndexes))
	for i, libIndex := range libIndexes {
		if libIndex < 0 || int(libIndex) >= len(db.data.Library) {
			return []LibraryEntry{}, fmt.Errorf(
				"library anime index (%d) does not exist",
				libIndex,
			)
		}
		entries[i] = db.data.Library[libIndex]
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

func (db *Database) AddAnime(entry LibraryEntry) error {
	db.data.Library = append(db.data.Library, entry)
	return db.Save()
}

func (db *Database) RemoveAnimeById(id string) error {
	for i, entry := range db.data.Library {
		if entry.ID == id {
			db.data.Library = slices.Delete(db.data.Library, i, i+1)
		}
	}
	return db.Save()
}

func (db *Database) RemoveAnimeByIndex(index LibraryIndex) error {
	db.data.Library = slices.Delete(db.data.Library, int(index), int(index)+1)
	return db.Save()
}

func (db *Database) UpdateAnime(i LibraryIndex, entry LibraryEntry) error {
	db.data.Library[i] = entry
	return db.Save()
}

func (db Database) Save() error {
	if !db.isLoaded {
		return fmt.Errorf("database was not initialized properly")
	}
	bytes, err := msgpack.Marshal(db.data)
	if err != nil {
		return err
	}
	os.WriteFile(dbFileName, bytes, 0o644)
	return nil
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
