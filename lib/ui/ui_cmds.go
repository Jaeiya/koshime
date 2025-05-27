package ui

import (
	"github.com/Jaeiya/koshime/lib/database"
	"github.com/Jaeiya/koshime/lib/kitsu"
	"github.com/charmbracelet/bubbles/v2/list"
	tea "github.com/charmbracelet/bubbletea/v2"
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

func (s AnimeFinderSource) String() string {
	switch s {
	case Kitsu:
		return "Kitsu🌐"
	case Local:
		return "Local📁"
	default:
		return "Unknown"
	}
}

type AnimeFinderResult struct {
	ListItems []list.Item
	InfoItems []AnimeInfo
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
	info := make([]AnimeInfo, len(anime))
	items := make([]list.Item, len(anime))
	for i, item := range anime {
		items[i] = NewListItem(
			item.Attributes.CanonicalTitle,
			item.Attributes.Titles.English,
			i,
		)
		info[i] = AnimeInfo{
			ID:        item.ID,
			JpnTitle:  item.Attributes.CanonicalTitle,
			EngTitle:  item.Attributes.Titles.English,
			AltTitles: item.Attributes.AltTitles,
			ShowType:  item.Attributes.Type,
			Status:    item.Attributes.Status,
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

	info := make([]AnimeInfo, len(anime))
	items := make([]list.Item, len(anime))
	for i, item := range anime {
		items[i] = NewListItem(item.JPN_Title, item.ENG_Title, i)
		info[i] = AnimeInfo{
			ID:        item.ID,
			LibID:     item.LibID,
			JpnTitle:  item.JPN_Title,
			EngTitle:  item.ENG_Title,
			AltTitles: item.AltTitles,
			ShowType:  string(item.Type),
			Status:    item.Status,
			Synopsis:  item.Synopsis,
			Progress:  item.Progress,
			Episodes:  item.Episodes,
			Slug:      item.Slug,
		}
	}

	return AnimeFinderResult{items, info}, nil
}

func FindKitsuAnime(query string, maxResults int, status []kitsu.AnimeStatus) tea.Cmd {
	return func() tea.Msg {
		af := NewKitsuAnimeFinder(maxResults, status)
		results, err := af.Search(query)
		if err != nil {
			return err
		}
		return results
	}
}

func FindLocalAnime(query string, maxResults int, db *database.Database) tea.Cmd {
	return func() tea.Msg {
		af := NewLocalAnimeFinder(maxResults, db)
		results, err := af.Search(query)
		if err != nil {
			return err
		}
		return results
	}
}
