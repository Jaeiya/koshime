package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/Jaeiya/koshime/internal/database"
	"github.com/Jaeiya/koshime/internal/kitsu"
	"github.com/Jaeiya/koshime/internal/logger"
	"github.com/Jaeiya/koshime/internal/qbittorrent"
	"github.com/Jaeiya/koshime/internal/utils"
)

const watchedDir = "(watched)"

type WatchState int

const (
	_           = WatchState(iota)
	Pilot       // 00 episodes
	Watched     // Already watched episodes
	NonSeasonal // Episodes that are not following seasonal numbering
	Mismatched
)

var (
	ErrDropAnimeFailed = errors.New("failed to drop anime")
	ErrSaveProgress    = errors.New("an error occurred while saving anime")
)

type Progress struct {
	LastEp      int
	NextEp      int
	IsCompleted bool
}

func WatchedDir() string {
	return filepath.Join(utils.FileSys{}.WorkingDir(), watchedDir)
}

func CreateWatchedDir() error {
	logger.Log(logger.Debug, "creating watched dir")
	if _, err := os.Stat(WatchedDir()); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err = os.Mkdir(WatchedDir(), 0o750); err != nil {
				return err
			}
			return nil
		}
		return err
	}
	return nil
}

func DeleteWatchedDir() error {
	logger.Log(logger.Debug, "deleting watched dir")
	if err := os.Remove(WatchedDir()); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return nil
}

func MoveFansubFile(anime FilteredAnime) error {
	animeFile := anime.FileInfo.Filename
	watchPath := filepath.Join(WatchedDir(), animeFile)
	fs := utils.FileSys{}
	if err := fs.MoveFile(animeFile, watchPath); err != nil {
		return fmt.Errorf("failed to move fansub file: %w", err)
	}
	return nil
}

// ListWorkingAnime returns a list of all anime that potentially match an
// existing fansub file in the working directory. The threshold
// is the similarity percent required to match.
func ListWorkingAnime(db *database.Database, threshold int) ([]FilteredAnime, error) {
	fs := utils.FileSys{}
	stream, err := fs.NewFilenameStream(fs.WorkingDir())
	if err != nil {
		return nil, fmt.Errorf("failed to list working anime: %w", err)
	}

	ff := FansubFilter{}
	items, err := ff.FilterByAnime(stream, db.Anime(), threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to list working anime: %w", err)
	}

	slices.SortFunc(items, func(a, b FilteredAnime) int {
		return CompareAnime(a.Value, b.Value)
	})

	return items, nil
}

func PlayAnime(fileName string) error {
	fs := utils.FileSys{}
	var cmd *exec.Cmd

	filePath := filepath.Join(fs.WorkingDir(), fileName)
	if !fs.FileExists(filePath) {
		return fmt.Errorf("failed to play file: file not found")
	}

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/C", "start", "", filePath)
	case "darwin":
		cmd = exec.Command("open", filePath)
	default:
		cmd = exec.Command("xdg-open", filePath)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to play file: %w", err)
	}
	return nil
}

func SaveProgress(
	db *database.Database,
	anime FilteredAnime,
	state WatchState,
) (Progress, error) {
	fs := utils.FileSys{}
	if !fs.FileExists(filepath.Join(fs.WorkingDir(), anime.FileInfo.Filename)) {
		return Progress{}, fmt.Errorf(
			"%w: fansub file has been moved or deleted",
			ErrSaveProgress,
		)
	}

	p := Progress{LastEp: anime.Value.Progress}

	// 🟡 This only works for fansubs that follow seasonal episode counts.
	// A seasonal count means that each new season's first episode
	// starts at 1.
	if state == Watched || state == Pilot {
		if err := MoveFansubFile(anime); err != nil {
			return Progress{}, fmt.Errorf("%w: %w", ErrSaveProgress, err)
		}
		return p, nil
	}

	/*
	 * INFO: We assume the user is downloading anime in the order they want to
	 * watch it, therefore no matter what the file episode says, we update
	 * to next episode number. This allows support for non-seasonal episode
	 * counts.
	 */
	p.NextEp = p.LastEp + 1

	progResp, err := kitsu.UpdateAnimeProgress(
		anime.Value.LibID,
		db.Profile().AccessToken,
		p.NextEp,
	)
	if err != nil {
		return Progress{}, fmt.Errorf("%w: %w", ErrSaveProgress, err)
	}

	// 🟢 Kitsu does not always know the correct total episodes for a series
	// until the series is about to end.
	anime.Value.Episodes = progResp.Included[0].Attributes.EpisodeCount

	// 🟢 When an anime is completed (progress updated to match total episodes), the
	// anime status is automatically updated by Kitsu, unless the episode count
	// is unknown (0).
	if p.NextEp == anime.Value.Episodes {
		if err = CompleteAnime(db, anime.Value.LibID); err != nil {
			return p, fmt.Errorf("%w: %w", ErrSaveProgress, err)
		}
		p.IsCompleted = true
		if err = MoveFansubFile(anime); err != nil {
			return p, fmt.Errorf("%w: %w", ErrSaveProgress, err)
		}
		return p, nil
	}

	anime.Value.Progress = p.NextEp
	if err = db.UpdateAnime(anime.Value); err != nil {
		return p, fmt.Errorf("%w: %w", ErrSaveProgress, err)
	}

	if err = MoveFansubFile(anime); err != nil {
		return p, fmt.Errorf("%w: %w", ErrSaveProgress, err)
	}
	return p, nil
}

// DropAnime sets the status of an anime to 'dropped', deletes the
// anime from the local database, and attempts to remove an
// assigned qBittorrent feed.
func DropAnime(db *database.Database, libID string) error {
	p := db.Profile()
	logger.Log(logger.Debug, "DropAnime(): finding anime")
	anime, ok := db.FindAnimeByLibId(libID)
	if !ok {
		return fmt.Errorf("failed to find anime to drop: %w", ErrDropAnimeFailed)
	}
	logger.Log(logger.Debug, "DropAnime(): removing feed")
	if err := RemoveFeed(p.QbtPort, anime.QbtFeed.Name); err != nil {
		return fmt.Errorf("failed to remove dropped anime feed: %w", err)
	}
	logger.Log(logger.Debug, "DropAnime(): setting anime status to dropped")
	if err := kitsu.DropAnime(libID, p.AccessToken); err != nil {
		return ErrDropAnimeFailed
	}
	logger.Log(logger.Debug, "DropAnime(): deleting anime from database")
	if err := db.DeleteAnime(libID); err != nil {
		return fmt.Errorf("failed to delete anime while dropping: %w", err)
	}
	return nil
}

// CompleteAnime sets the status of an anime to 'completed', deletes
// the anime from the local database, and attempts to remove an
// assigned qBittorrent feed.
func CompleteAnime(db *database.Database, libID string) error {
	p := db.Profile()
	logger.Log(logger.Debug, "CompleteAnime(): finding anime")
	anime, ok := db.FindAnimeByLibId(libID)
	if !ok {
		return fmt.Errorf("failed to find anime to complete")
	}
	logger.Log(logger.Debug, "CompleteAnime(): removing anime feed")
	if err := RemoveFeed(p.QbtPort, anime.QbtFeed.Name); err != nil {
		return fmt.Errorf("failed to delete completed anime feed: %w", err)
	}
	logger.Log(logger.Debug, "CompleteAnime(): setting anime status to completed")
	if _, err := kitsu.SetAnimeStatus(libID, p.AccessToken, kitsu.LibAnimeCompleted); err != nil {
		return err
	}
	logger.Log(logger.Debug, "CompleteAnime(): deleting anime from database")
	if err := db.DeleteAnime(libID); err != nil {
		return fmt.Errorf("failed to delete completed anime: %w", err)
	}
	return nil
}

// DeleteAnime deletes an anime from the users Kitsu library
// and local database.
func DeleteAnime(db *database.Database, libID string) error {
	p := db.Profile()
	logger.Log(logger.Debug, "DeleteAnime(): finding anime")
	anime, ok := db.FindAnimeByLibId(libID)
	if !ok {
		return fmt.Errorf("failed to find anime to delete")
	}
	logger.Log(logger.Debug, "DeleteAnime(): removing feed")
	if err := RemoveFeed(p.QbtPort, anime.QbtFeed.Name); err != nil {
		return fmt.Errorf("failed to remove deleted anime feed: %w", err)
	}
	logger.Log(logger.Debug, "DeleteAnime(): deleting anime from kitsu library")
	if _, err := kitsu.DeleteAnime(libID, p.AccessToken); err != nil {
		return fmt.Errorf("failed to delete anime from library: %w", err)
	}
	logger.Log(logger.Debug, "DeleteAnime(): deleting anime from database")
	if err := db.DeleteAnime(libID); err != nil {
		return fmt.Errorf("failed to delete anime from database: %w", err)
	}
	return nil
}

// DeleteFansub searches the working directory and filters all fansub
// files, deleting the ones that have the highest score match.
func DeleteFansub(anime kitsu.Anime) error {
	fs := utils.FileSys{}
	ff := FansubFilter{}
	logger.Log(logger.Debug, "DeleteFansub(): getting working file stream")
	stream, err := fs.NewFilenameStream(fs.WorkingDir())
	if err != nil {
		return fmt.Errorf("failed get file list for deletion: %w", err)
	}
	logger.Log(logger.Debug, "DeleteFansub(): filtering file names by anime")
	fileNames, err := ff.FilterFilenamesByAnime(anime, stream, 33)
	if err != nil {
		return fmt.Errorf("failed to filter files for deletion: %w", err)
	}
	logger.Log(logger.Debug, "DeleteFansub(): deleting fansub files")
	for _, file := range fileNames {
		path := filepath.Join(fs.WorkingDir(), file)
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to delete fansub file: %w", err)
		}
	}
	return nil
}

// RemoveFeed attempts to remove an existing Anime feed if
// the user has setup qBittorrent.
func RemoveFeed(port int, feedName string) error {
	if port > 0 && feedName != "" {
		logger.Log(logger.Debug, "RemoveFeed(): logging into qbt")
		qbt, err := qbittorrent.NewLogin(strconv.Itoa(port))
		if err != nil {
			return err
		}
		logger.Log(logger.Debug, "RemoveFeed(): deleting feed")
		if err = qbt.DeleteFeed(feedName); err != nil {
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

	logger.Log(logger.Debug, "UpdateFeeds(): logging into qbt")
	qb, err := qbittorrent.NewLogin(strconv.Itoa(db.Profile().QbtPort))
	if err != nil {
		return fmt.Errorf("failed to login to update feeds: %w", err)
	}

	logger.Log(logger.Debug, "UpdateFeeds(): loading feeds")
	feeds, err := qb.Feeds()
	if err != nil {
		return fmt.Errorf("failed to load feeds to update: %w", err)
	}

	anime := slices.DeleteFunc(db.Anime(), func(a kitsu.Anime) bool {
		return len(a.QbtFeed.Name) > 0
	})
	updatedAnime := make([]kitsu.Anime, 0, len(feeds))

	logger.Log(
		logger.Debug,
		"UpdateFeeds(): updating [%d] anime against [%d] feeds",
		len(anime),
		len(feeds),
	)
	for _, feed := range feeds {
		head, tail, found := strings.Cut(feed.Name, "(")
		if !found { // All feeds added by Koshime have open/closing paren
			continue
		}
		engTitle := strings.TrimSpace(head)
		jpnTitle := strings.TrimSpace(strings.TrimSuffix(tail, ")"))

		for i, a := range anime {
			hasEngTitle := len(a.ENG_Title) > 0 && strings.EqualFold(a.ENG_Title, engTitle)
			hasJpnTitle := len(a.JPN_Title) > 0 && strings.EqualFold(a.JPN_Title, jpnTitle)
			if hasEngTitle || hasJpnTitle {
				a.QbtFeed.Name = feed.Name
				a.QbtFeed.RuleURI = feed.URL
				updatedAnime = append(updatedAnime, a)
				anime = slices.Delete(anime, i, i+1)
				break
			}
		}
	}

	return db.UpdateAllAnime(updatedAnime)
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
