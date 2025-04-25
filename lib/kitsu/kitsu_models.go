package kitsu

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
