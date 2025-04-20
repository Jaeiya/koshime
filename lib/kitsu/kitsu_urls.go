package kitsu

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const (
	apiAuthTokenURL = "https://kitsu.app/api/oauth/token"
	apiLibraryURL   = "https://kitsu.app/api/edge/library-entries"
	apiAnimeURL     = "https://kitsu.app/api/edge/anime"
	apiUsersURL     = "https://kitsu.app/api/edge/users"
)

type URLType int

const (
	LibraryURL = URLType(iota)
	AnimeURL
	UserURL
)

type AnimeType string

const (
	TV      = AnimeType("tv")
	Movie   = AnimeType("movie")
	OVA     = AnimeType("ova")
	ONA     = AnimeType("ona")
	Music   = AnimeType("music")
	Special = AnimeType("special")
)

type MediaType string

const (
	MangaMedia = MediaType("manga")
	AnimeMedia = MediaType("anime")
)

type DataCategory string

const (
	AnimeCategory = DataCategory("anime")
	StatsCategory = DataCategory("stats")
)

type AnimeField string

const (
	AbbreviatedTitlesField = AnimeField("abbreviatedTitles")
	AverageRatingField     = AnimeField("averageRating")
	CanonicalTitleField    = AnimeField("canonicalTitle")
	EpisodeCountField      = AnimeField("episodeCount")
	SlugField              = AnimeField("slug")
	SynopsisField          = AnimeField("synopsis")
	StartDate              = AnimeField("startDate")
	TitlesField            = AnimeField("titles")
	AgeRatingField         = AnimeField("ageRating")
)

type AnimeStatus string

const (
	// Currently airing or going to be airing in the future
	//
	//🟡 Kitsu does not always update "upcoming" to "current"
	// on time, so use this if you want to catch anime
	// that should be airing within the season.
	NewAnime = AnimeStatus(CurrentAnime + "," + UpcomingAnime)
	// Currently Airing
	CurrentAnime = AnimeStatus("current")
	// Going to be airing at some point in the future
	UpcomingAnime = AnimeStatus("upcoming")
	// To be announced
	TBAAnime = AnimeStatus("tba")
	// Confirmed but not released
	UnreleasedAnime = AnimeStatus("unreleased")
	// Finished airing
	FinishedAnime = AnimeStatus("finished")
)

type LibraryAnimeStatus string

const (
	CurrentLibAnime   = LibraryAnimeStatus("current")
	CompletedLibAnime = LibraryAnimeStatus("completed")
	DroppedLibAnime   = LibraryAnimeStatus("dropped")
	PlannedLibAnime   = LibraryAnimeStatus("planned")
)

func GetAnimeInfoURL(query string, status AnimeStatus, maxItems int) (KitsuURL, error) {
	u, err := NewKitsuURL(AnimeURL)
	if err != nil {
		return KitsuURL{}, nil
	}

	if maxItems > 200 {
		return KitsuURL{}, fmt.Errorf("max items cannot be larger than 200")
	}

	return u.QueryText(query).
		PageLimit(maxItems).
		QueryAnimeType(TV).
		QueryAnimeStatus(status).
		QueryAnimeFields(
			[]AnimeField{
				TitlesField,
				CanonicalTitleField,
				AbbreviatedTitlesField,
				AverageRatingField,
				EpisodeCountField,
				SlugField,
				SynopsisField,
				AgeRatingField,
			},
		), nil
}

// GetAnimeLibInfoURL requires at least one user library
// anime ID and returns all relevant information for
// that entry.
func GetAnimeLibInfoURL(libIDs []string) (KitsuURL, error) {
	u, err := NewKitsuURL(LibraryURL)
	if err != nil {
		return KitsuURL{}, nil
	}

	return u.QueryIDs(libIDs).
		IncludeCategory([]DataCategory{AnimeCategory}).
		QueryAnimeFields([]AnimeField{
			EpisodeCountField,
			AverageRatingField,
			SynopsisField,
		}).
		PageLimit(len(libIDs)), nil
}

func GetUserLibAnimeURL(userID string, status LibraryAnimeStatus) (KitsuURL, error) {
	u, err := NewKitsuURL(UserURL)
	if err != nil {
		return KitsuURL{}, err
	}

	// Point to users library specifically
	u.url = u.url.JoinPath(userID, "library-entries")

	return u.QueryLibAnimeStatus(status).
			PageLimit(200).
			IncludeCategory([]DataCategory{AnimeCategory}),
		nil
}

type KitsuURL struct {
	url *url.URL
}

func (k KitsuURL) String() string {
	return k.url.String()
}

func NewKitsuURL(uType URLType) (KitsuURL, error) {
	var u *url.URL
	var err error

	switch uType {
	case LibraryURL:
		u, err = url.Parse(apiLibraryURL)
	case AnimeURL:
		u, err = url.Parse(apiAnimeURL)
	case UserURL:
		u, err = url.Parse(apiUsersURL)
	}

	if u == nil {
		e := fmt.Errorf("invalid URL type")
		if err != nil {
			e = err
		}
		return KitsuURL{}, e
	}

	return KitsuURL{u}, err
}

func (k KitsuURL) QueryText(text string) KitsuURL {
	q := k.url.Query()
	q.Add("filter[text]", text)
	k.url.RawQuery = q.Encode()
	return k
}

func (k KitsuURL) PageLimit(limit int) KitsuURL {
	q := k.url.Query()
	q.Add("page[limit]", strconv.Itoa(limit))
	k.url.RawQuery = q.Encode()
	return k
}

func (k KitsuURL) QueryUserName(name string) KitsuURL {
	q := k.url.Query()
	q.Add("filter[name]", name)
	k.url.RawQuery = q.Encode()
	return k
}

func (k KitsuURL) QueryUserID(id string) KitsuURL {
	q := k.url.Query()
	q.Add("filter[user_id]", id)
	k.url.RawQuery = q.Encode()
	return k
}

func (k KitsuURL) QueryIDs(ids []string) KitsuURL {
	q := k.url.Query()
	q.Add("filter[id]", strings.Join(ids, ","))
	k.url.RawQuery = q.Encode()
	return k
}

func (k KitsuURL) QueryAnimeType(aType AnimeType) KitsuURL {
	q := k.url.Query()
	q.Add("filter[subtype]", string(aType))
	k.url.RawQuery = q.Encode()
	return k
}

func (k KitsuURL) QueryAnimeStatus(status AnimeStatus) KitsuURL {
	return k.queryStatus(string(status))
}

func (k KitsuURL) QueryLibAnimeStatus(status LibraryAnimeStatus) KitsuURL {
	return k.queryStatus(string(status))
}

func (k KitsuURL) queryStatus(status string) KitsuURL {
	q := k.url.Query()
	q.Add("filter[status]", string(status))
	k.url.RawQuery = q.Encode()
	return k
}

func (k KitsuURL) QueryMediaType(mType string) KitsuURL {
	q := k.url.Query()
	q.Add("filter[kind]", string(mType))
	k.url.RawQuery = q.Encode()
	return k
}

func (k KitsuURL) IncludeCategory(cats []DataCategory) KitsuURL {
	q := k.url.Query()

	var sb strings.Builder
	sb.Grow(5*len(cats) + 1)

	for i, c := range cats {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(string(c))
	}

	q.Add("include", sb.String())
	k.url.RawQuery = q.Encode()
	return k
}

func (k KitsuURL) QueryAnimeFields(fields []AnimeField) KitsuURL {
	q := k.url.Query()

	var sb strings.Builder
	sb.Grow(13 * len(fields))

	for i, f := range fields {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(string(f))
	}

	q.Add("fields[anime]", sb.String())
	k.url.RawQuery = q.Encode()
	return k
}

func GetProfileURL(userName string) (string, error) {
	u, err := url.Parse(apiUsersURL)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Add("filter[name]", userName)
	q.Add("include", "stats")

	u.RawQuery = q.Encode()
	return u.String(), nil
}
