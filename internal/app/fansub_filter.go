package app

import (
	"fmt"
	"strings"

	"github.com/Jaeiya/koshime/internal/database"
	"github.com/Jaeiya/koshime/internal/kitsu"
	"github.com/Jaeiya/koshime/internal/utils"
)

type FilteredAnime struct {
	Anime    kitsu.Anime
	FileInfo FansubFileInfo
	Score    int
}

type AnimeTitleMap map[string]map[string]struct{}

type FansubFilter struct{}

// All parses all possible fansubs from the provided stream and
// returns their info.
func (ff FansubFilter) All(stream utils.FilenameIterator) ([]FansubFileInfo, error) {
	fp := FansubParser{}
	fansubs := []FansubFileInfo{}

	for {
		fileName, ok := stream.Next()
		if !ok {
			break
		}

		if !fp.IsSupported(fileName) {
			continue
		}

		fansub, err := fp.Parse(fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to parse filename: %w", err)
		}

		fansubs = append(fansubs, fansub)
	}

	return fansubs, nil
}

// FilterByLibEntry returns a slice of anime data by how closely a file name
// can be matched against an anime in the users library. A score of greater
// than 50 is required to be considered a match.
//
// 🔵 Redundant episodic file names are ignored; file names will
// match in ascending order: '04' matches before '06'
func (ff FansubFilter) FilterByLibEntry(
	stream utils.FilenameIterator,
	db *database.Database,
) ([]FilteredAnime, error) {
	fp := FansubParser{}
	animeLib := db.Anime()
	animeWordMap := ff.buildAnimeWordMap(animeLib)
	foundStore := map[string]struct{}{}
	fileFoundStore := map[string]struct{}{}
	filteredAnime := []FilteredAnime{}

	for {
		fileName, ok := stream.Next()
		if !ok {
			break
		}

		if !fp.IsSupported(fileName) {
			continue
		}

		fansub, err := fp.Parse(fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to parse filename: %w", err)
		}

		// Do not try score the same fansub file
		if _, exists := fileFoundStore[fansub.Title]; exists {
			continue
		}

		libID, bindingExists := db.FindFileBinding(fansub.Title)
		if bindingExists {
			entry, entryExists := db.FindAnimeByLibId(libID)
			if !entryExists {
				return nil, fmt.Errorf("missing anime in library with ID: %s", libID)
			}

			if _, exists := foundStore[entry.ID]; exists {
				continue
			}

			filteredAnime = append(filteredAnime, FilteredAnime{
				Anime:    entry,
				FileInfo: fansub,
				Score:    100,
			})

			foundStore[entry.ID] = struct{}{}
			continue
		}

		found := FilteredAnime{}
		titleWords := strings.Fields(ff.normalizeTitle(fansub.Title))

		for _, anime := range animeLib {
			if _, exists := foundStore[anime.ID]; exists {
				continue
			}
			confidence := 0
			for _, word := range titleWords {
				if _, exists := animeWordMap[anime.ID][word]; exists {
					confidence++
				}
			}
			score := int(float64(confidence) / float64(len(titleWords)) * 100)
			if score > found.Score {
				found.Score = score
				found.Anime = anime
				found.FileInfo = fansub
			}
		}

		if found.Score > 50 {
			foundStore[found.Anime.ID] = struct{}{}
			fileFoundStore[fansub.Title] = struct{}{}
			filteredAnime = append(filteredAnime, found)
		}

	}
	return filteredAnime, nil
}

func (ff FansubFilter) FilterFilenamesByAnime(
	anime kitsu.Anime,
	stream utils.FilenameIterator,
) (map[int][]string, error) {
	fp := FansubParser{}
	found := map[int][]string{}
	titleWordMap := ff.buildWordsFromTitles(anime)

	for {
		fileName, ok := stream.Next()
		if !ok {
			break
		}

		if !fp.IsSupported(fileName) {
			continue
		}

		fansub, err := fp.Parse(fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to parse filename: %w", err)
		}

		titleWords := strings.Fields(ff.normalizeTitle(fansub.Title))
		score := 0
		confidence := 0

		for _, word := range titleWords {
			if _, exists := titleWordMap[word]; exists {
				confidence++
			}
			score = int(float64(confidence) / float64(len(titleWords)) * 100)
		}
		score += ff.scoreTitles(fansub.Title, anime)

		if score > 50 {
			if len(titleWords) <= len(titleWordMap) {
				score -= len(titleWordMap) - len(titleWords)
			}

			if v, exists := found[score]; exists {
				found[score] = append(v, fansub.Filename)
			} else {
				found[score] = []string{fansub.Filename}
			}
		}
	}
	return found, nil
}

func (ff FansubFilter) buildAnimeWordMap(entries []kitsu.Anime) AnimeTitleMap {
	animeWordMap := make(map[string]map[string]struct{}, len(entries))
	for _, entry := range entries {
		animeWordMap[entry.ID] = ff.buildWordsFromTitles(entry)
	}
	return animeWordMap
}

func (ff FansubFilter) buildWordsFromTitles(anime kitsu.Anime) map[string]struct{} {
	titleTokenMap := map[string]struct{}{}
	var titles []string
	if anime.JPN_Title != "" {
		titles = append(titles, ff.normalizeTitle(anime.JPN_Title))
	}
	if anime.ENG_Title != "" {
		titles = append(titles, ff.normalizeTitle(anime.ENG_Title))
	}
	for _, title := range anime.AltTitles {
		titles = append(titles, ff.normalizeTitle(title))
	}
	for _, title := range titles {
		for token := range strings.FieldsSeq(title) {
			titleTokenMap[token] = struct{}{}
		}
	}
	return titleTokenMap
}

func (ff FansubFilter) scoreTitles(title string, anime kitsu.Anime) int {
	title = ff.normalizeTitle(title)
	jpnTitle := ff.normalizeTitle(anime.JPN_Title)
	engTitle := ff.normalizeTitle(anime.ENG_Title)

	if jpnTitle == title || engTitle == title {
		return 1
	}

	for _, altTitle := range anime.AltTitles {
		altTitle = ff.normalizeTitle(altTitle)
		if altTitle == title {
			return 1
		}
	}

	return 0
}

func (FansubFilter) normalizeTitle(title string) string {
	return strings.ToLower(
		strings.TrimSpace(
			strings.ReplaceAll(utils.ReplaceCutset(title, ".-,?:![]()<>", " "), "  ", " "),
		),
	)
}
