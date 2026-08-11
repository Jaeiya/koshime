package app

import (
	"charm.land/bubbles/v2/list"
	"github.com/Jaeiya/koshime/internal/database"
	"github.com/Jaeiya/koshime/internal/kitsu"
	"github.com/Jaeiya/koshime/internal/ui"
	"github.com/Jaeiya/koshime/internal/utils"
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
	case NoSource:
		return "None", "⛔"
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
	InfoItems []kitsu.Anime
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
	info := make([]kitsu.Anime, len(anime))
	items := make([]list.Item, len(anime))
	for i, item := range anime {
		items[i] = ui.NewListItem(
			item.Attributes.CanonicalTitle,
			item.Attributes.Titles.English,
			i,
		)

		info[i] = kitsu.Anime{
			ID:        item.ID,
			JPN_Title: item.Attributes.CanonicalTitle,
			ENG_Title: item.Attributes.Titles.English,
			AltTitles: item.Attributes.AltTitles,
			Type:      item.Attributes.Type,
			Status:    item.Attributes.Status,
			Synopsis:  item.Attributes.Synopsis,
			AvgRating: utils.CalcRating(item.Attributes.AvgRating),
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

	info := make([]kitsu.Anime, len(anime))
	items := make([]list.Item, len(anime))
	for i, item := range anime {
		items[i] = ui.NewListItem(item.JPN_Title, item.ENG_Title, i)
		info[i] = kitsu.Anime{
			ID:        item.ID,
			LibID:     item.LibID,
			JPN_Title: item.JPN_Title,
			ENG_Title: item.ENG_Title,
			AltTitles: item.AltTitles,
			Type:      item.Type,
			Status:    item.Status,
			Synopsis:  item.Synopsis,
			Progress:  item.Progress,
			AvgRating: utils.CalcRating(item.AvgRating),
			Episodes:  item.Episodes,
			Slug:      item.Slug,
		}
	}

	return AnimeFinderResult{items, info}, nil
}
