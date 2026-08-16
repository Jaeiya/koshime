package app

import (
	"fmt"
	"slices"
	"strings"

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
	animeList []kitsu.Anime,
	scoreThreshold int,
) ([]FilteredAnime, error) {
	fp := FansubParser{}
	animeFoundStore := map[string]FilteredAnime{}
	fileFoundStore := map[string]struct{}{}
	var filteredAnime []FilteredAnime

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
			s := ff.score(fansub.Title, anime, scoreThreshold)
			if s > found.Score {
				found.Anime = anime
				found.FileInfo = fansub
				found.Score = s
			}
		}

		if found.Score > 0 {
			if f, exists := animeFoundStore[found.Anime.ID]; exists {
				if found.Score > f.Score {
					animeFoundStore[found.Anime.ID] = found
				}
			} else {
				animeFoundStore[found.Anime.ID] = found
			}
		}

		fileFoundStore[fansub.Title] = struct{}{}
	}

	for _, foundAnime := range animeFoundStore {
		filteredAnime = append(filteredAnime, foundAnime)
	}

	return filteredAnime, nil
}

func (ff FansubFilter) FilterFilenamesByAnime(
	anime kitsu.Anime,
	stream utils.FilenameIterator,
	scoreThreshold int,
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

		s := ff.score(fansub.Title, anime, scoreThreshold)
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

// score determines a match percentage based on how well the
// specified title matches any of the anime's titles.
//
// 🔵 A match score <= 50 is considered a 0% match. This is
// because a score of <= 50 cannot ever be guaranteed as a
// reasonable match in this context.
func (ff FansubFilter) score(title string, anime kitsu.Anime, threshold int) int {
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
	if score > threshold {
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

	if score > threshold {
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
