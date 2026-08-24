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
	"sync"
	"time"

	"github.com/Jaeiya/koshime/internal/kitsu"
	"github.com/Jaeiya/koshime/internal/utils"
	"github.com/shamaton/msgpack/v2"
)

var (
	fileSys    utils.FileSys
	dbFilePath = func() string {
		return filepath.Join(fileSys.GetWorkingDir(), "koshime.db")
	}()
	flateWriterPool = sync.Pool{
		New: func() any {
			w, _ := flate.NewWriter(io.Discard, flate.DefaultCompression)
			return w
		},
	}
)

type Data struct {
	Profile kitsu.Profile
	Library []kitsu.Anime
}

type Database struct {
	data Data
	rw   DbRWriter
}

type DbRWriter interface {
	Write(filePath string, data []byte) error
	Read(filePath string) ([]byte, error)
	Exists() bool
}

type dbRWriter struct{}

func (dbRWriter) Read(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

func (dbRWriter) Write(filePath string, data []byte) error {
	return os.WriteFile(filePath, data, 0o600)
}

func (dbRWriter) Exists() bool {
	return fileSys.FileExists(dbFilePath)
}

func NewDatabase(rw DbRWriter) (*Database, error) {
	db := &Database{}
	db.rw = dbRWriter{}
	if rw != nil {
		db.rw = rw
	}

	if !db.rw.Exists() {
		return db, nil
	}

	fileBytes, err := db.rw.Read(dbFilePath)
	if err != nil {
		return nil, err
	}
	uncompressed, err := decompressData(fileBytes)
	if err != nil {
		return nil, err
	}
	err = msgpack.Unmarshal(uncompressed, &db.data)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (db Database) Anime() []kitsu.Anime {
	return slices.Clone(db.data.Library)
}

func (db Database) Profile() kitsu.Profile {
	return db.data.Profile
}

func (db Database) Exists() bool {
	return db.rw.Exists()
}

// LoadData overwrites all existing data and saves it.
//
// 🟠 This is destructive and should only be used to
// bootstrap or refresh the database.
func (db *Database) LoadData(d Data) error {
	db.data = d
	return db.Save()
}

// LoadProfile overwrites existing profile data and saves it.
//
// 🟠 This is destructive and should only be used to
// bootstrap or refresh the user profile.
func (db *Database) LoadProfile(p kitsu.Profile) error {
	db.data.Profile = p
	return db.Save()
}

// LoadLibrary overwrites existing library data and saves it.
//
// 🟠 This is destructive and should only be used to
// bootstrap or refresh the library.
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
	idx := db.LibraryIndex(libID)
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
	idx := db.LibraryIndex(libID)
	if idx < 0 {
		return fmt.Errorf("failed to delete anime: library id not found [%s]", libID)
	}
	db.data.Library = slices.Delete(db.data.Library, idx, idx+1)
	return db.Save()
}

func (db *Database) UpdateAnime(anime kitsu.Anime) error {
	idx := db.LibraryIndex(anime.LibID)
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
		idx := db.LibraryIndex(anime.LibID)
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

// LibraryIndex looks up a library entry by its library id and returns
// its index or -1 if it was not found.
func (db *Database) LibraryIndex(libID string) int {
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
	return db.LoadProfile(db.data.Profile)
}

func (db Database) Save() error {
	bytes, err := msgpack.Marshal(db.data)
	if err != nil {
		return err
	}
	compressed, err := compressData(bytes)
	if err != nil {
		return err
	}
	err = db.rw.Write(dbFilePath, compressed)
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
	b := bytes.NewBuffer(make([]byte, 0, len(data)))
	fw, _ := flateWriterPool.Get().(*flate.Writer)
	defer flateWriterPool.Put(fw)

	fw.Reset(b)

	_, err := fw.Write(data)
	if err != nil {
		return nil, err
	}
	if err := fw.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func decompressData(data []byte) ([]byte, error) {
	reader := flate.NewReader(bytes.NewReader(data))
	defer reader.Close()

	var out bytes.Buffer
	if _, err := io.Copy(&out, reader); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}
