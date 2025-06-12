package lib

import (
	"fmt"
	"strconv"

	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/kitsu"
	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/charmbracelet/bubbles/v2/list"
)

type AnimeFinder interface {
	Search(query string) (AnimeFinderResult, error)
}

type AnimeFinderSource int

const (
	NoSource = AnimeFinderSource(iota)
	Kitsu
	Local
)

// Name returns the stringified version of the source, as well
// as its associated emoji.
func (s AnimeFinderSource) Name() (string, string) {
	switch s {
	case Kitsu:
		return "Kitsu", "🌐"
	case Local:
		return "Local", "📁"
	default:
		return "Unknown", ""
	}
}

type AnimeFinderResult struct {
	ListItems []list.Item
	InfoItems []ui.AnimeInfo
}

type KitsuAnimeSearch struct {
	maxResults int
	status     []kitsu.AnimeStatus
}

func NewKitsuAnimeFinder(maxResults int, status []kitsu.AnimeStatus) KitsuAnimeSearch {
	return KitsuAnimeSearch{maxResults, status}
}

func (af KitsuAnimeSearch) Search(query string) (AnimeFinderResult, error) {
	anime, err := kitsu.FindAnime(query, af.status, af.maxResults)
	if err != nil {
		return AnimeFinderResult{}, err
	}
	info := make([]ui.AnimeInfo, len(anime))
	items := make([]list.Item, len(anime))
	for i, item := range anime {
		items[i] = ui.NewListItem(
			item.Attributes.CanonicalTitle,
			item.Attributes.Titles.English,
			i,
		)

		info[i] = ui.AnimeInfo{
			ID:        item.ID,
			JpnTitle:  item.Attributes.CanonicalTitle,
			EngTitle:  item.Attributes.Titles.English,
			AltTitles: item.Attributes.AltTitles,
			ShowType:  item.Attributes.Type,
			Status:    item.Attributes.Status,
			Synopsis:  item.Attributes.Synopsis,
			Progress:  -1,
			AvgRating: calcRating(item.Attributes.AvgRating),
			Episodes:  item.Attributes.EpCount,
			Slug:      item.Attributes.Slug,
		}
	}

	return AnimeFinderResult{items, info}, nil
}

type LocalAnimeSearch struct {
	db         *database.Database
	maxResults int
}

func NewLocalAnimeFinder(maxResults int, db *database.Database) LocalAnimeSearch {
	return LocalAnimeSearch{db, maxResults}
}

func (af LocalAnimeSearch) Search(query string) (AnimeFinderResult, error) {
	anime, err := af.db.FindAnime(query)
	if err != nil {
		return AnimeFinderResult{}, err
	}

	anime = anime[:min(af.maxResults, len(anime))]

	info := make([]ui.AnimeInfo, len(anime))
	items := make([]list.Item, len(anime))
	for i, item := range anime {
		items[i] = ui.NewListItem(item.JPN_Title, item.ENG_Title, i)
		info[i] = ui.AnimeInfo{
			ID:        item.ID,
			LibID:     item.LibID,
			JpnTitle:  item.JPN_Title,
			EngTitle:  item.ENG_Title,
			AltTitles: item.AltTitles,
			ShowType:  string(item.Type),
			Status:    item.Status,
			Synopsis:  item.Synopsis,
			Progress:  item.Progress,
			AvgRating: calcRating(item.AvgRating),
			Episodes:  item.Episodes,
			Slug:      item.Slug,
		}
	}

	return AnimeFinderResult{items, info}, nil
}

func calcRating(r string) string {
	if r == "" {
		return ""
	}
	rawRating, err := strconv.ParseFloat(r, 64)
	if err != nil {
		panic(fmt.Errorf("could not calc avg rating: %w", err))
	}
	return fmt.Sprintf("%.2f", rawRating/10)
}
