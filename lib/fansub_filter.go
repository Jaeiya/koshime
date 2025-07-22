package lib

import (
	"fmt"
	"strings"

	"github.com/Jaeiya/koshime/lib/kitsu"
	"github.com/Jaeiya/koshime/lib/utils"
)

type FilteredAnime struct {
	LibEntry kitsu.LibraryEntry
	FileInfo FansubFileInfo
	Score    int
}

type AnimeTitleMap map[string]map[string]struct{}

type FansubFilter struct{}

func (ff FansubFilter) FilterByLibEntry(
	stream utils.FilenameIterator,
	animeLibrary []kitsu.LibraryEntry,
) ([]FilteredAnime, error) {
	fp := FansubParser{}
	animeWordMap := ff.buildAnimeWordMap(animeLibrary)
	foundStore := map[string]struct{}{}
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

		found := FilteredAnime{}
		titleWords := strings.Fields(ff.normalizeTitle(fansub.Title))

		for _, anime := range animeLibrary {
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
				found.LibEntry = anime
				found.FileInfo = fansub
			}

			if found.Score >= 50 {
				foundStore[anime.ID] = struct{}{}
				break
			}
		}

		if found.Score >= 50 {
			filteredAnime = append(filteredAnime, found)
		}

	}
	return filteredAnime, nil
}

func (ff FansubFilter) buildAnimeWordMap(entries []kitsu.LibraryEntry) AnimeTitleMap {
	animeWordMap := make(map[string]map[string]struct{}, len(entries))

	for _, entry := range entries {
		titleTokenMap := map[string]struct{}{}
		var titles []string
		if entry.JPN_Title != "" {
			titles = append(titles, ff.normalizeTitle(entry.JPN_Title))
		}
		if entry.ENG_Title != "" {
			titles = append(titles, ff.normalizeTitle(entry.ENG_Title))
		}
		for _, title := range entry.AltTitles {
			titles = append(titles, ff.normalizeTitle(title))
		}
		for _, title := range titles {
			for token := range strings.FieldsSeq(title) {
				titleTokenMap[token] = struct{}{}
			}
		}
		animeWordMap[entry.ID] = titleTokenMap
	}

	return animeWordMap
}

func (FansubFilter) normalizeTitle(title string) string {
	return strings.ToLower(
		strings.TrimSpace(
			strings.ReplaceAll(utils.ReplaceCutset(title, ".-,?:!", " "), "  ", " "),
		),
	)
}
