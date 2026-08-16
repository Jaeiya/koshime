package app

import (
	"fmt"
	"slices"
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

func (ff FansubFilter) FilterByLibEntry2(
	stream utils.FilenameIterator,
	animeList []kitsu.Anime,
) ([]FilteredAnime, error) {
	fp := FansubParser{}
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
		// Only score unique fansub titles
		if _, exists := fileFoundStore[fansub.Title]; exists {
			continue
		}

		found := FilteredAnime{}
		for _, anime := range animeList {
			s := ff.score(fansub.Title, anime)
			if s > found.Score {
				found.Anime = anime
				found.FileInfo = fansub
				found.Score = s
			}
		}

		if found.Score > 0 {
			filteredAnime = append(filteredAnime, found)
		}

		fileFoundStore[fansub.Title] = struct{}{}
	}

	return filteredAnime, nil
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
) ([]string, error) {
	fp := FansubParser{}
	found := map[int][]string{}
	topScore := 0

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

		s := ff.score(fansub.Title, anime)
		if s > topScore {
			topScore = s
		}

		if v, exists := found[s]; exists {
			found[s] = append(v, fansub.Filename)
		} else {
			found[s] = []string{fansub.Filename}
		}
	}
	return found[topScore], nil
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

// score determines a match percentage based on how well the
// specified title matches any of the anime's titles.
//
// 🔵 A match score <= 50 is considered a 0% match. This is
// because a score of <= 50 cannot ever be guaranteed as a
// reasonable match in this context.
func (ff FansubFilter) score(title string, anime kitsu.Anime) int {
	titleWords := strings.Fields(ff.normalizeTitle(title))
	if len(titleWords) == 0 {
		return 0
	}

	jpnWords := strings.Fields(ff.normalizeTitle(anime.JPN_Title))
	engWords := strings.Fields(ff.normalizeTitle(anime.ENG_Title))

	altWordSlices := make([][]string, len(anime.AltTitles))
	for i, t := range anime.AltTitles {
		altWordSlices[i] = strings.Fields(ff.normalizeTitle(t))
	}

	wordSlices := append([][]string{jpnWords, engWords}, altWordSlices...)

	// ========== EXACT MATCH ==========
	for _, slice := range wordSlices {
		if slices.Equal(slice, titleWords) {
			return 100
		}
	}

	score := 0

	// ========= SUBSTR WORD MATCH ==========
	for _, wordSlice := range wordSlices {
		if utils.AreWordsInSlice(wordSlice, titleWords) {
			s := len(titleWords) * 100 / len(wordSlice)
			if s > score {
				score = s
			}
		}
	}

	// A fuzzy search is not necessary if we found an
	// acceptable score, because a substr word match
	// is a more accurate match.
	if score > 50 {
		return score
	}

	// ========== FUZZY WORD MATCH ==========
	for _, words := range wordSlices {
		wordMap := ff.buildWordMap(words)
		if len(wordMap) == 0 {
			continue
		}
		foundWords := map[string]struct{}{}
		matchCount := 0
		for _, word := range titleWords {
			if _, exists := foundWords[word]; exists {
				continue
			}
			if _, exists := wordMap[word]; exists {
				matchCount += 1
				foundWords[word] = struct{}{}
			}
		}
		s := matchCount * 100 / len(wordMap)
		if len(wordMap) < len(titleWords) {
			s = matchCount * 100 / len(titleWords)
		}
		if s > score {
			score = s
		}
	}

	if score > 50 {
		return score
	}
	return 0
}

func (ff FansubFilter) buildWordMap(titleWords []string) map[string]struct{} {
	wordMap := map[string]struct{}{}
	if len(titleWords) == 0 {
		return wordMap
	}
	for _, word := range titleWords {
		wordMap[word] = struct{}{}
	}
	return wordMap
}

func (FansubFilter) normalizeTitle(title string) string {
	return strings.ToLower(
		strings.TrimSpace(
			strings.ReplaceAll(utils.ReplaceCutset(title, ".-,?:![]()<>", " "), "  ", " "),
		),
	)
}
