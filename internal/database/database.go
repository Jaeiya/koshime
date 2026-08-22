package database

import (
	"bytes"
	"compress/flate"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Jaeiya/koshime/internal/kitsu"
	"github.com/Jaeiya/koshime/internal/utils"
	"github.com/shamaton/msgpack/v2"
)

var (
	fileSys    utils.FileSys
	dbFileName = "koshime.db"
)

type Data struct {
	Profile kitsu.Profile
	Library []kitsu.Anime
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

func (db Database) Anime() []kitsu.Anime {
	return slices.Clone(db.data.Library)
}

func (db Database) Profile() kitsu.Profile {
	return db.data.Profile
}

func (db *Database) Exists() bool {
	dbPath := filepath.Join(fileSys.GetWorkingDir(), dbFileName)
	return fileSys.FileExists(dbPath)
}

// Load an existing database. In order to initialize the database
// properly, you can either pass the data directly to the
// constructor or use this function to load it from file.
func (db *Database) Load() error {
	if db.isLoaded {
		return nil
	}
	fileBytes, err := os.ReadFile(filepath.Join(fileSys.GetWorkingDir(), dbFileName))
	if err != nil {
		return err
	}
	uncompressed, err := decompressData(fileBytes)
	if err != nil {
		return err
	}
	err = msgpack.Unmarshal(uncompressed, &db.data)
	if err != nil {
		return err
	}
	db.isLoaded = true
	return nil
}

// LoadData overwrites existing data and saves it to file.
//
// 🟠 This is destructive and should only be used
// to initialize an empty database.
func (db *Database) LoadData(d Data) error {
	db.data = d
	db.isLoaded = true
	return db.Save()
}

// LoadLibrary overwrites existing library with new entries.
//
// 🟠 This is destructive and should only be used
// if the library is in an invalid state.
func (db *Database) LoadLibrary(entries []kitsu.Anime) error {
	db.data.Library = entries
	return db.Save()
}

// FindAnime fuzzy-finds an anime by searching all titles & synopsis
// then returns all possible matches.
func (db Database) FindAnime(query string) ([]kitsu.Anime, error) {
	query = strings.ToLower(query)

	var libIndexes []int
	for i, entry := range db.data.Library {
		if hasMatchingEntry(entry, query) {
			libIndexes = append(libIndexes, i)
		}
	}

	if len(libIndexes) == 0 {
		return []kitsu.Anime{}, nil
	}

	anime := make([]kitsu.Anime, len(libIndexes))
	for i, idx := range libIndexes {
		anime[i] = db.data.Library[idx]
	}

	return anime, nil
}

func (db Database) FindAnimeByLibId(libID string) (kitsu.Anime, bool) {
	idx := db.LibraryLookup(libID)
	if idx < 0 {
		return kitsu.Anime{}, false
	}
	return db.data.Library[idx], true
}

func (db *Database) AddAnime(entry kitsu.Anime) error {
	db.data.Library = append(db.data.Library, entry)
	return db.Save()
}

func (db *Database) DeleteAnime(libID string) error {
	idx := db.LibraryLookup(libID)
	if idx < 0 {
		return fmt.Errorf("failed to delete anime: library id not found [%s]", libID)
	}
	db.data.Library = slices.Delete(db.data.Library, idx, idx+1)
	return db.Save()
}

func (db *Database) UpdateAnime(anime kitsu.Anime) error {
	idx := db.LibraryLookup(anime.LibID)
	if idx < 0 {
		return fmt.Errorf("library id not found for: %s", anime.ENG_Title)
	}
	db.data.Library[idx] = anime
	err := db.Save()
	if err != nil {
		return fmt.Errorf("failed to save updated anime: %w", err)
	}
	return nil
}

func (db *Database) UpdateAllAnime(animeList []kitsu.Anime) error {
	for _, anime := range animeList {
		idx := db.LibraryLookup(anime.LibID)
		if idx < 0 {
			return fmt.Errorf("library id not found for: %s", anime.ENG_Title)
		}
		db.data.Library[idx] = anime
	}
	err := db.Save()
	if err != nil {
		return fmt.Errorf("failed to save updated anime: %w", err)
	}
	return nil
}

// LibraryLookup looks up an entry by its library id and returns
// its index or -1 if it was not found.
func (db *Database) LibraryLookup(libID string) int {
	for i, entry := range db.data.Library {
		if entry.LibID == libID {
			return i
		}
	}
	return -1
}

func (db *Database) SaveTokenData(token, refreshToken string, expiresIn int) error {
	db.data.Profile.AccessToken = token
	db.data.Profile.RefreshToken = refreshToken
	/*
		INFO: Because this is a duration and not a time stamp, we stamp
		it ourselves by adding the unix time.
	*/
	db.data.Profile.TokenExpirationSec = int64(expiresIn) + time.Now().Unix()
	return db.SaveProfile(db.data.Profile)
}

func (db *Database) SaveProfile(p kitsu.Profile) error {
	db.data.Profile = p
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
	compressed, err := compressData(bytes)
	if err != nil {
		return err
	}
	err = os.WriteFile(dbFileName, compressed, 0o600)
	if err != nil {
		return err
	}
	return nil
}

func hasMatchingEntry(e kitsu.Anime, q string) bool {
	if strings.Contains(strings.ToLower(e.ENG_Title), q) ||
		strings.Contains(strings.ToLower(e.JPN_Title), q) ||
		strings.Contains(strings.ToLower(e.Synopsis), q) {
		return true
	}
	return slices.ContainsFunc(e.AltTitles, func(s string) bool {
		return strings.Contains(strings.ToLower(s), q)
	})
}

func compressData(data []byte) ([]byte, error) {
	var b bytes.Buffer
	writer, err := flate.NewWriter(&b, flate.BestCompression)
	if err != nil {
		return nil, err
	}
	_, err = writer.Write(data)
	if err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func decompressData(data []byte) ([]byte, error) {
	reader := flate.NewReader(bytes.NewReader(data))
	defer reader.Close()

	var out bytes.Buffer
	//nolint:gosec // DoS bomb impossible since this isn't a web service
	if _, err := io.Copy(&out, reader); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}
