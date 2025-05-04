package kitsu

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const KitsuDomain = "https://kitsu.app"

type APIUrl string

const (
	apiAuthTokenURL = APIUrl(KitsuDomain + "/api/oauth/token")
	apiLibraryURL   = APIUrl(KitsuDomain + "/api/edge/library-entries")
	apiAnimeURL     = APIUrl(KitsuDomain + "/api/edge/anime")
	apiUsersURL     = APIUrl(KitsuDomain + "/api/edge/users")
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
	AnimeNew = AnimeStatus(AnimeAiring + "," + AnimeUpcoming)
	// Currently Airing
	AnimeAiring = AnimeStatus("current")
	// Going to be airing at some point in the future
	AnimeUpcoming = AnimeStatus("upcoming")
	// To be announced
	AnimeTBA = AnimeStatus("tba")
	// Confirmed but not released
	AnimeUnreleased = AnimeStatus("unreleased")
	// Finished airing
	AnimeFinished = AnimeStatus("finished")
)

type LibAnimeStatus string

const (
	// This is the "Currently Watching" library filter in Kitsu
	LibAnimeWatching  = LibAnimeStatus("current")
	LibAnimeCompleted = LibAnimeStatus("completed")
	LibAnimeDropped   = LibAnimeStatus("dropped")
	LibAnimePlanned   = LibAnimeStatus("planned")
	LibAnimeOnHold    = LibAnimeStatus("on_hold")
)

func getLibEntryURL(libID string) APIUrl {
	u, _ := url.JoinPath(string(apiLibraryURL), libID)
	return APIUrl(u)
}

func getAnimeInfoQURL(query string, status AnimeStatus, maxItems int) (APIUrl, error) {
	u, err := newQURL(AnimeURL)
	if err != nil {
		return "", nil
	}

	if maxItems > 200 || maxItems < 1 {
		return "", fmt.Errorf("max items should be between 1 and 200; inclusive")
	}

	u = u.QueryText(query).
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
		)

	return APIUrl(u.Build()), nil
}

// getAnimeLibInfoQURL requires at least one user library
// anime ID and returns all relevant information for
// that entry.
func getAnimeLibInfoQURL(libIDs []string) (APIUrl, error) {
	u, err := newQURL(LibraryURL)
	if err != nil {
		return "", nil
	}

	u = u.QueryIDs(libIDs).
		IncludeCategory([]DataCategory{AnimeCategory}).
		QueryAnimeFields([]AnimeField{
			EpisodeCountField,
			AverageRatingField,
			SynopsisField,
		}).
		PageLimit(len(libIDs))

	return APIUrl(u.Build()), nil
}

func getUserLibAnimeQURL(userID string, status LibAnimeStatus) (APIUrl, error) {
	u, err := newQURL(UserURL)
	if err != nil {
		return "", err
	}

	// Point to users library specifically
	u.url = u.url.JoinPath(userID, "library-entries")

	u = u.QueryLibAnimeStatus(status).
		PageLimit(200).
		IncludeCategory([]DataCategory{AnimeCategory})

	return APIUrl(u.Build()), nil
}

func getProfileQURL(userName string) (APIUrl, error) {
	u, err := newQURL(UserURL)
	if err != nil {
		return "", err
	}

	u = u.QueryUserName(userName).
		IncludeCategory([]DataCategory{StatsCategory}).
		PageLimit(1)

	return APIUrl(u.Build()), nil
}

type KitsuURL struct {
	url   *url.URL
	query url.Values
}

// newQURL returns a new Kitsu Query URL.
func newQURL(uType URLType) (*KitsuURL, error) {
	var u *url.URL
	var err error

	switch uType {
	case LibraryURL:
		u, err = url.Parse(string(apiLibraryURL))
	case AnimeURL:
		u, err = url.Parse(string(apiAnimeURL))
	case UserURL:
		u, err = url.Parse(string(apiUsersURL))
	}

	if u == nil {
		e := fmt.Errorf("invalid URL type")
		if err != nil {
			e = err
		}
		return &KitsuURL{}, e
	}

	return &KitsuURL{u, u.Query()}, err
}

func (k *KitsuURL) QueryText(text string) *KitsuURL {
	k.query.Add("filter[text]", text)
	return k
}

func (k *KitsuURL) PageLimit(limit int) *KitsuURL {
	k.query.Add("page[limit]", strconv.Itoa(limit))
	return k
}

func (k *KitsuURL) QueryUserName(name string) *KitsuURL {
	k.query.Add("filter[name]", name)
	return k
}

func (k *KitsuURL) QueryUserID(id string) *KitsuURL {
	k.query.Add("filter[user_id]", id)
	return k
}

func (k *KitsuURL) QueryIDs(ids []string) *KitsuURL {
	k.query.Add("filter[id]", strings.Join(ids, ","))
	return k
}

func (k *KitsuURL) QueryAnimeType(aType AnimeType) *KitsuURL {
	k.query.Add("filter[subtype]", string(aType))
	return k
}

func (k *KitsuURL) QueryAnimeStatus(status AnimeStatus) *KitsuURL {
	return k.queryStatus(string(status))
}

func (k *KitsuURL) QueryLibAnimeStatus(status LibAnimeStatus) *KitsuURL {
	return k.queryStatus(string(status))
}

func (k *KitsuURL) queryStatus(status string) *KitsuURL {
	k.query.Add("filter[status]", string(status))
	return k
}

func (k *KitsuURL) QueryMediaType(mType string) *KitsuURL {
	k.query.Add("filter[kind]", string(mType))
	return k
}

func (k *KitsuURL) IncludeCategory(cats []DataCategory) *KitsuURL {
	var sb strings.Builder
	sb.Grow(5*len(cats) + 1)

	for i, c := range cats {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(string(c))
	}

	k.query.Add("include", sb.String())
	return k
}

func (k *KitsuURL) QueryAnimeFields(fields []AnimeField) *KitsuURL {
	var sb strings.Builder
	sb.Grow(13 * len(fields))

	for i, f := range fields {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(string(f))
	}

	k.query.Add("fields[anime]", sb.String())
	return k
}

func (k *KitsuURL) Build() string {
	k.url.RawQuery = k.query.Encode()
	return k.url.String()
}
