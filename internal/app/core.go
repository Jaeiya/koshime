package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Jaeiya/koshime/internal/database"
	"github.com/Jaeiya/koshime/internal/kitsu"
	"github.com/Jaeiya/koshime/internal/qbittorrent"
	"github.com/Jaeiya/koshime/internal/utils"
)

var ErrDropAnimeFailed = errors.New("failed to drop anime")

// DropAnime sets the status of an anime to 'dropped', deletes the
// anime from the local database, and attempts to remove an
// assigned qBittorrent feed.
func DropAnime(db *database.Database, libID string) error {
	p := db.Profile()
	anime, ok := db.FindAnimeByLibId(libID)
	if !ok {
		return fmt.Errorf("failed to find anime to drop: %w", ErrDropAnimeFailed)
	}
	if err := RemoveFeed(p.QbtPort, anime.QbtFeed.Name); err != nil {
		return fmt.Errorf("failed to remove dropped anime feed: %w", err)
	}
	if err := kitsu.DropAnime(libID, p.AccessToken); err != nil {
		return ErrDropAnimeFailed
	}
	if err := db.DeleteAnimeById(libID); err != nil {
		return fmt.Errorf("failed to delete anime while dropping: %w", err)
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

func FuzzyFindAnime(anime []kitsu.Anime, search string) (kitsu.Anime, bool) {
	ff := FansubFilter{}
	topAnime := kitsu.Anime{}
	topScore := 0

	for _, a := range anime {
		s := ff.Score(search, a, 5)
		if s > topScore {
			topScore = s
			topAnime = a
		}
	}

	if topScore == 0 {
		return topAnime, false
	}

	return topAnime, true
}

func UpdateFeeds(db *database.Database) error {
	if db.Profile().QbtPort == 0 {
		return nil
	}

	qb, err := qbittorrent.NewLogin(strconv.Itoa(db.Profile().QbtPort))
	if err != nil {
		return fmt.Errorf("failed to login to update feeds: %w", err)
	}

	feeds, err := qb.Feeds()
	if err != nil {
		return fmt.Errorf("failed to load feeds to update: %w", err)
	}

	for _, a := range db.Anime() {
		for _, feed := range feeds {
			// All feeds added by Koshime have a paren
			if !strings.Contains(feed.Name, "(") {
				continue
			}
			titles := strings.Split(feed.Name, "(")
			titles[0] = strings.TrimSpace(titles[0])
			titles[1] = strings.TrimSpace(titles[1])
			if strings.EqualFold(titles[0], a.ENG_Title) ||
				strings.EqualFold(titles[1][:len(titles[1])-1], a.JPN_Title) {
				a.QbtFeed.Name = feed.Name
				a.QbtFeed.RuleURI = feed.URL
				err := db.UpdateAnime(a)
				if err != nil {
					return fmt.Errorf("failed to update feeds: %w", err)
				}
				break
			}
		}
	}

	return nil
}

func CompareAnime(animeA, animeB kitsu.Anime) int {
	aTitle := animeA.ENG_Title
	if aTitle == "" {
		aTitle = animeA.JPN_Title
	}
	bTitle := animeB.ENG_Title
	if bTitle == "" {
		bTitle = animeB.JPN_Title
	}
	return strings.Compare(aTitle, bTitle)
}
