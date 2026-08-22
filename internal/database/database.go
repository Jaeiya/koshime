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

type LibraryIndex int

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

func (db Database) FindAnime(query string) ([]kitsu.Anime, error) {
	indexes, err := db.FindLibAnimeIndex(query)
	if err != nil {
		return []kitsu.Anime{}, err
	}

	if len(indexes) == 0 {
		return []kitsu.Anime{}, nil
	}

	return db.AnimeByIndex(indexes...)
}

// FindLibAnimeIndex uses the query to do a partial lookup against
// all available anime titles, including alt titles, and returns
// the library index of all matches found.
func (db Database) FindLibAnimeIndex(query string) ([]LibraryIndex, error) {
	query = strings.ToLower(query)
	var indexes []LibraryIndex
	for i, entry := range db.data.Library {
		if hasMatchingEntry(entry, query) {
			indexes = append(indexes, LibraryIndex(i))
		}
	}
	return indexes, nil
}

func (db Database) FindAnimeByLibId(id string) (kitsu.Anime, bool) {
	for _, entry := range db.data.Library {
		if entry.LibID == id {
			return entry, true
		}
	}
	return kitsu.Anime{}, false
}

// AnimeByIndex will retrieve library anime using the specified
// library indexes.
func (db Database) AnimeByIndex(libIndexes ...LibraryIndex) ([]kitsu.Anime, error) {
	if len(libIndexes) == 0 {
		return []kitsu.Anime{}, fmt.Errorf("missing indexes to lookup")
	}
	entries := make([]kitsu.Anime, len(libIndexes))
	for i, libIndex := range libIndexes {
		if libIndex < 0 || int(libIndex) >= len(db.data.Library) {
			return []kitsu.Anime{}, fmt.Errorf(
				"library anime index (%d) does not exist",
				libIndex,
			)
		}
		entries[i] = db.data.Library[libIndex]
	}
	return entries, nil
}

func (db Database) Anime() []kitsu.Anime {
	return slices.Clone(db.data.Library)
}

func (db Database) Profile() kitsu.Profile {
	return db.data.Profile
}

// SaveProfile overwrites the existing profile with
// the specified one.
func (db *Database) SaveProfile(p kitsu.Profile) error {
	db.data.Profile = p
	return db.Save()
}

// SaveLibrary overwrites the existing library with
// the specified one.
func (db *Database) SaveLibrary(p []kitsu.Anime) error {
	db.data.Library = p
	return db.Save()
}

func (db *Database) AddAnime(entry kitsu.Anime) error {
	db.data.Library = append(db.data.Library, entry)
	return db.Save()
}

func (db *Database) DeleteAnimeById(libID string) error {
	var deleted bool
	for i, entry := range db.data.Library {
		if entry.LibID == libID {
			db.data.Library = slices.Delete(db.data.Library, i, i+1)
			deleted = true
			break
		}
	}
	if !deleted {
		return fmt.Errorf("could not find anime-library id [%s]", libID)
	}
	return db.Save()
}

func (db *Database) DeleteAnimeByIndex(index LibraryIndex) error {
	db.data.Library = slices.Delete(db.data.Library, int(index), int(index)+1)
	return db.Save()
}

func (db *Database) UpdateAnimeAt(i LibraryIndex, entry kitsu.Anime) error {
	db.data.Library[i] = entry
	return db.Save()
}

func (db *Database) UpdateAnime(anime kitsu.Anime) error {
	idx, found := db.LibraryLookup(anime.LibID)
	if !found {
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
		idx, found := db.LibraryLookup(anime.LibID)
		if !found {
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

// LibraryLookup looks up an anime by library id and returns
// its library index and whether or not it was found.
func (db *Database) LibraryLookup(libID string) (int, bool) {
	for i, entry := range db.data.Library {
		if entry.LibID == libID {
			return i, true
		}
	}
	return 0, false
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
