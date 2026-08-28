package database

import (
	"os"
	"testing"

	"github.com/Jaeiya/koshime/internal/kitsu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeDbRWriter implements DbRWriter in memory.
type fakeDbRWriter struct {
	files  map[string][]byte
	exists bool
}

func newFakeRWriter() *fakeDbRWriter {
	return &fakeDbRWriter{
		files: make(map[string][]byte),
	}
}

func (f *fakeDbRWriter) Read(filePath string) ([]byte, error) {
	data, ok := f.files[filePath]
	if !ok {
		return nil, os.ErrNotExist
	}
	return data, nil
}

func (f *fakeDbRWriter) Write(filePath string, data []byte) error {
	f.files[filePath] = data
	f.exists = true
	return nil
}

func (f *fakeDbRWriter) Exists() bool {
	return f.exists
}

func TestDatabaseNew(t *testing.T) {
	t.Parallel()
	rw := newFakeRWriter()
	db, err := NewDatabase(rw)
	require.NoError(t, err)
	assert.False(t, db.Exists())
}

func TestDatabaseLoadData(t *testing.T) {
	t.Parallel()
	rw := newFakeRWriter()
	db, err := NewDatabase(rw)
	require.NoError(t, err)

	expectedID := "1234"
	expectedLib := []kitsu.Anime{
		{ID: "1234"},
		{ID: "4321"},
		{ID: "5678"},
	}
	expectedData := Data{
		Profile: kitsu.Profile{ID: expectedID},
		Library: expectedLib,
	}

	err = db.LoadData(expectedData)
	require.NoError(t, err)
	assert.Equal(t, expectedData, db.data, "overwrites existing data")

	loadedDb, err := NewDatabase(rw)
	require.NoError(t, err)
	assert.Equal(t, expectedData, loadedDb.data, "loads existing data")
}

func TestDatabaseAddAnime(t *testing.T) {
	t.Parallel()
	rw := newFakeRWriter()
	db, err := NewDatabase(rw)
	require.NoError(t, err)
	db.data.Library = []kitsu.Anime{
		{ID: "4321"},
	}

	expAnime := kitsu.Anime{ID: "1234"}
	err = db.AddAnime(expAnime)
	require.NoError(t, err)

	assert.Equal(t, expAnime, db.data.Library[1], "anime should be appended to existing lib")

	loadedDb, err := NewDatabase(rw)
	require.NoError(t, err)
	assert.Equal(t, db.data, loadedDb.data, "loaded database should reflect added anime")
}

func TestDatabaseDeleteAnime(t *testing.T) {
	t.Parallel()
	rw := newFakeRWriter()
	db, err := NewDatabase(rw)
	require.NoError(t, err)
	db.data.Library = []kitsu.Anime{
		{ID: "4321"},
	}

	expAnime := []kitsu.Anime{
		{LibID: "1234"},
		{LibID: "4321"},
		{LibID: "5678"},
	}
	db.data.Library = expAnime

	err = db.DeleteAnime("5678")
	require.NoError(t, err)
	assert.Equal(t, expAnime[:2], db.data.Library)

	err = db.DeleteAnime("1234")
	require.NoError(t, err)
	assert.Equal(t, expAnime[0], db.data.Library[0])
	assert.Len(t, db.data.Library, 1)

	loadedDb, err := NewDatabase(rw)
	require.NoError(t, err)
	assert.Equal(t, db.data, loadedDb.data, "loaded database should reflect deleted anime")
}

func TestDatabaseUpdateAnime(t *testing.T) {
	t.Parallel()
	rw := newFakeRWriter()
	db, err := NewDatabase(rw)
	require.NoError(t, err)

	db.data.Library = []kitsu.Anime{
		{LibID: "1234", ID: "01234"},
		{LibID: "4321", ID: "04321"},
		{LibID: "5678", ID: "05678"},
	}
	expAnime := kitsu.Anime{
		LibID:     "4321",
		ENG_Title: "hello world",
	}
	err = db.UpdateAnime(expAnime)
	require.NoError(t, err)
	assert.Equal(t, expAnime, db.data.Library[1], "anime is updated")

	loadedDb, err := NewDatabase(rw)
	require.NoError(t, err)
	assert.Equal(t, db.data, loadedDb.data, "loaded database should reflect updated anime")
}

func TestDatabaseUpdateAllAnime(t *testing.T) {
	t.Parallel()
	rw := newFakeRWriter()
	db, err := NewDatabase(rw)
	require.NoError(t, err)

	db.data.Library = []kitsu.Anime{
		{LibID: "1234", ID: "01234"},
		{LibID: "4321", ID: "04321"},
		{LibID: "5678", ID: "05678"},
	}
	expAnime := []kitsu.Anime{
		{LibID: "4321", ENG_Title: "test title 1"},
		{LibID: "1234", ENG_Title: "test title 2"},
		{LibID: "5678", ENG_Title: "test title 3"},
	}
	err = db.UpdateAllAnime(expAnime)
	require.NoError(t, err)
	assert.ElementsMatch(t, expAnime, db.data.Library, "anime are updated")

	loadedDb, err := NewDatabase(rw)
	require.NoError(t, err)
	assert.Equal(
		t,
		db.data.Library,
		loadedDb.data.Library,
		"loaded database should reflect updated anime",
	)
}
