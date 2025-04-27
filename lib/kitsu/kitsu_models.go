package kitsu

import "encoding/json"

type AuthToken struct {
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
		ID         string `json:"id"`
		Attributes struct {
			Name     string `json:"name"`
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

type Anime struct {
	ID         string `json:"id"`
	Attributes struct {
		CanonicalTitle string `json:"canonicalTitle"`
		Titles         struct {
			English string `json:"en"`
			Romaji  string `json:"en_jp"`
		} `json:"titles"`
		AltTitles []string `json:"abbreviatedTitles"`
		AvgRating string   `json:"averageRating"`
		AgeRating string   `json:"ageRating"`
		StartDate string   `json:"startDate"`
		EndDate   string   `json:"endDate"`
		Slug      string   `json:"slug"`
		Synopsis  string   `json:"synopsis"`
	} `json:"attributes"`
}

type LibraryAnime struct {
	Data []struct {
		LibID      string `json:"id"`
		Attributes struct {
			Progress     int    `json:"progress"`
			ProgressedAt string `json:"progressedAt"`
			StartedAt    string `json:"startedAt"`
		} `json:"attributes"`
	} `json:"data"`

	Included []Anime `json:"included"`
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
