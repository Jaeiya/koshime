package views

import (
	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/kitsu"
	"github.com/Jaeiya/koshime/lib/ui"
	"github.com/charmbracelet/bubbles/v2/list"
)

type AnimeFinder interface {
	Search(query string) (AnimeFinderResult, error)
}

type AnimeFinderResult struct {
	listItems []list.Item
	infoItems []ui.AnimeInfo
}

type KitsuAnimeFinder struct {
	maxResults int
	status     []kitsu.AnimeStatus
}

func NewKitsuAnimeFinder(maxResults int, status []kitsu.AnimeStatus) KitsuAnimeFinder {
	return KitsuAnimeFinder{maxResults, status}
}

func (af KitsuAnimeFinder) Search(query string) (AnimeFinderResult, error) {
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
			JpnTitle:  item.Attributes.CanonicalTitle,
			EngTitle:  item.Attributes.Titles.English,
			AltTitles: item.Attributes.AltTitles,
			ShowType:  item.Attributes.Type,
			Synopsis:  item.Attributes.Synopsis,
			Progress:  -1,
			Episodes:  item.Attributes.EpCount,
			Slug:      item.Attributes.Slug,
		}
	}

	return AnimeFinderResult{items, info}, nil
}

type LocalAnimeFinder struct {
	db         *database.Database
	maxResults int
}

func NewLocalAnimeFinder(maxResults int, db *database.Database) LocalAnimeFinder {
	return LocalAnimeFinder{db, maxResults}
}

func (af LocalAnimeFinder) Search(query string) (AnimeFinderResult, error) {
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
			JpnTitle:  item.JPN_Title,
			EngTitle:  item.ENG_Title,
			AltTitles: item.AltTitles,
			ShowType:  string(item.Type),
			Synopsis:  item.Synopsis,
			Progress:  item.Progress,
			Episodes:  item.Episodes,
			Slug:      item.Slug,
		}
	}

	return AnimeFinderResult{items, info}, nil
}
