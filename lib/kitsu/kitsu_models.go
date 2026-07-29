package kitsu

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type LibraryEntry struct {
	// Anime ID
	ID string
	// Anime User-library ID - Allows looking up User-specific Anime data
	LibID     string
	JPN_Title string
	ENG_Title string
	AltTitles []string
	Episodes  int
	Type      AnimeType
	Status    string
	Progress  int
	AvgRating string
	Synopsis  string
	Slug      string
}

type Profile struct {
	ID              string
	SecondsWatched  int
	CompletedSeries int
	Username        string
	Slug            string
	About           string
	Location        string
	Birthday        string
	Gender          string
	CreatedAt       string
	AccessToken     string
	RefreshToken    string
	QbtPort         int
	// Unix timestamp in seconds
	TokenExpirationSec int64
	// Unix timestamp in seconds
	LastUpdateSec int64
}

type AuthTokenData struct {
	Token     string `json:"access_token"`
	TokenType string `json:"token_type"`
	// Seconds until token expires
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	// Seconds since unix epoch
	CreatedAt int `json:"created_at"`
}

type ProfileData struct {
	Data []struct {
		ID string `json:"id"`
		// CreatedAt & Name are the only values that can by relied on.
		Attributes struct {
			Name     string `json:"name"`
			Slug     string `json:"slug"`
			About    string `json:"about"`
			Location string `json:"location"`
			Birthday string `json:"birthday"`
			Gender   string `json:"gender"`
			// An RFC 3339 (ISO 8601) formatted string
			CreatedAt string `json:"createdAt"`
		}
	} `json:"data"`

	Included []struct {
		Attributes struct {
			Stats struct {
				SecondsWatched int `json:"time"`
				CompletedAnime int `json:"completed"`
			} `json:"statsData"`
		}
	} `json:"included"`
}

type AnimeData struct {
	ID         string `json:"id"`
	Attributes struct {
		CanonicalTitle string `json:"canonicalTitle"`
		Titles         struct {
			English string `json:"en"` // Can be empty
			Romaji  string `json:"en_jp"`
		} `json:"titles"`
		// Can be empty
		AltTitles []string `json:"abbreviatedTitles"`
		// Is a float-percentage of favor (ex: 72.34) and can be empty
		AvgRating string `json:"averageRating"`
		// Can be empty
		AgeRating string `json:"ageRating"`
		EpCount   int    `json:"episodeCount"`
		StartDate string `json:"startDate"`
		EndDate   string `json:"endDate"`
		Type      string `json:"subtype"`
		Status    string `json:"status"`
		Slug      string `json:"slug"`
		Synopsis  string `json:"synopsis"`
	} `json:"attributes"`
}

type LibraryAnimeData struct {
	Data []struct {
		LibID      string `json:"id"`
		Attributes struct {
			Progress     int    `json:"progress"`
			ProgressedAt string `json:"progressedAt"`
			StartedAt    string `json:"startedAt"`
		} `json:"attributes"`
	} `json:"data"`

	Included []AnimeData `json:"included"`
}

type ProgressRespData struct {
	Data struct {
		Attributes struct {
			Progress int `json:"progress"`
		} `json:"attributes"`
	} `json:"data"`

	Included []struct {
		Attributes struct {
			EpisodeCount int `json:"episodeCount"`
		} `json:"attributes"`
	} `json:"included"`
}

type APIErrorData struct {
	Errors []struct {
		Status int    `json:"status"`
		Title  string `json:"title"`
		Detail string `json:"detail"`
	} `json:"errors"`
}

func (ed APIErrorData) String() string {
	var sb strings.Builder
	for _, err := range ed.Errors {
		errType := "ClientError"
		if err.Status >= 500 {
			errType = "ServerError"
		}
		detail := err.Detail
		if detail == "" {
			detail = err.Title
		}
		fmt.Fprintf(&sb, "%s: [HTTP %d] %s\n", errType, err.Status, detail)
	}
	if sb.Len() > 0 {
		return sb.String()
	}
	return "no error data"
}

type APIErrorDataV2 struct {
	Errors []struct {
		Status string `json:"status"`
		Title  string `json:"title"`
		Detail string `json:"detail"`
	} `json:"errors"`
}

func (ed APIErrorDataV2) String() string {
	var sb strings.Builder
	for _, err := range ed.Errors {
		errType := "ClientError"
		status, _ := strconv.Atoi(err.Status)
		if status >= 500 {
			errType = "ServerError"
		}
		detail := err.Detail
		if detail == "" {
			detail = err.Title
		}
		fmt.Fprintf(&sb, "%s: [HTTP %d] %s\n", errType, status, detail)
	}
	if sb.Len() > 0 {
		return sb.String()
	}
	return "no error data"
}

type AuthErrorData struct {
	Type        string `json:"error"`
	Description string `json:"error_description"`
}

func (aed AuthErrorData) String() string {
	return fmt.Sprintf("%s: %s", aed.Type, aed.Description)
}

func newAddAnimePayload(animeID, userID string, status LibAnimeStatus) ([]byte, error) {
	payload := struct {
		Data struct {
			Type       string `json:"type"`
			Attributes struct {
				Status string `json:"status"`
			} `json:"attributes"`
			Relationships struct {
				Anime struct {
					Data struct {
						Id   string `json:"id"`
						Type string `json:"type"`
					} `json:"data"`
				} `json:"anime"`
				User struct {
					Data struct {
						Id   string `json:"id"`
						Type string `json:"type"`
					} `json:"data"`
				} `json:"user"`
			} `json:"relationships"`
		} `json:"data"`
	}{}

	payload.Data.Type = "library-entries"
	payload.Data.Attributes.Status = string(status)
	payload.Data.Relationships.Anime.Data.Id = animeID
	payload.Data.Relationships.Anime.Data.Type = "anime"
	payload.Data.Relationships.User.Data.Id = userID
	payload.Data.Relationships.User.Data.Type = "users"

	return json.Marshal(payload)
}

func newAnimeStatusPayload(libID string, status LibAnimeStatus) ([]byte, error) {
	payload := struct {
		Data struct {
			Id         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Status string `json:"status"`
			} `json:"attributes"`
		} `json:"data"`
	}{}

	payload.Data.Id = libID
	payload.Data.Type = "library-entries"
	payload.Data.Attributes.Status = string(status)
	return json.Marshal(payload)
}

func newAnimeProgressPayload(libID string, progress int) ([]byte, error) {
	payload := struct {
		Data struct {
			Id         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Progress int `json:"progress"`
			} `json:"attributes"`
		} `json:"data"`
	}{}

	payload.Data.Id = libID
	payload.Data.Type = "library-entries"
	payload.Data.Attributes.Progress = progress
	return json.Marshal(payload)
}
