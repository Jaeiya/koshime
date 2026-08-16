package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Jaeiya/koshime/internal/database"
	"github.com/Jaeiya/koshime/internal/kitsu"
	"github.com/Jaeiya/koshime/internal/qbittorrent"
	"github.com/Jaeiya/koshime/internal/utils"
)

// DropAnime sets the status of an anime to 'dropped', deletes the
// anime from the local database, and attempts to remove an
// assigned qBittorrent feed.
func DropAnime(db *database.Database, libID string) error {
	p := db.Profile()
	anime, ok := db.FindAnimeByLibId(libID)
	if !ok {
		return fmt.Errorf("failed to find anime to drop")
	}
	if err := RemoveFeed(p.QbtPort, anime.QbtFeed.Name); err != nil {
		return fmt.Errorf("failed to delete dropped anime feed: %w", err)
	}
	if err := kitsu.DropAnime(libID, p.AccessToken); err != nil {
		return err
	}
	if err := db.DeleteAnimeById(libID); err != nil {
		return fmt.Errorf("failed to delete dropped anime: %w", err)
	}
	return nil
}

// CompleteAnime sets the status of an anime to 'completed', deletes
// the anime from the local database, and attempts to remove an
// assigned qBittorrent feed.
func CompleteAnime(db *database.Database, libID string) error {
	p := db.Profile()
	anime, ok := db.FindAnimeByLibId(libID)
	if !ok {
		return fmt.Errorf("failed to find anime to complete")
	}
	if err := RemoveFeed(p.QbtPort, anime.QbtFeed.Name); err != nil {
		return fmt.Errorf("failed to delete completed anime feed: %w", err)
	}
	_, err := kitsu.SetAnimeStatus(libID, p.AccessToken, kitsu.LibAnimeCompleted)
	if err != nil {
		return err
	}
	if err := db.DeleteAnimeById(libID); err != nil {
		return fmt.Errorf("failed to delete completed anime: %w", err)
	}
	return nil
}

// DeleteAnime deletes an anime from the users Kitsu library
// and local database.
func DeleteAnime(db *database.Database, libID string) error {
	p := db.Profile()
	_, err := kitsu.DeleteAnime(libID, p.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to delete anime from Library: %w", err)
	}
	err = db.DeleteAnimeById(libID)
	if err != nil {
		return fmt.Errorf("failed to delete anime from database: %w", err)
	}
	return nil
}

// DeleteFansub searches the working directory and filters all fansub
// files, deleting the ones that have the highest score match.
func DeleteFansub(anime kitsu.Anime) error {
	fs := utils.FileSys{}
	ff := FansubFilter{}
	stream, err := fs.NewFilenameStream(fs.GetWorkingDir())
	if err != nil {
		return fmt.Errorf("failed get file list for deletion: %w", err)
	}
	fileNames, err := ff.FilterFilenamesByAnime(anime, stream, 33)
	if err != nil {
		return fmt.Errorf("failed to filter files for deletion: %w", err)
	}
	for _, file := range fileNames {
		err := os.Remove(filepath.Join(fs.GetWorkingDir(), file))
		if err != nil {
			return fmt.Errorf("failed to delete fansub file: %w", err)
		}
	}
	return nil
}

// RemoveFeed attempts to remove an existing Anime feed if
// the user has setup qBittorrent.
func RemoveFeed(port int, feedName string) error {
	if port > 0 && feedName != "" {
		qbt, err := qbittorrent.NewLogin(strconv.Itoa(port))
		if err != nil {
			return err
		}
		err = qbt.DeleteFeed(feedName)
		if err != nil {
			return err
		}
	}
	return nil
}
