package kitsu

import "fmt"

const (
	apiAuthTokenURL = "https://kitsu.io/api/oauth/token"
	apiLibraryURL   = "https://kitsu.io/api/edge/library-entries"
	apiAnimeURL     = "https://kitsu.io/api/edge/anime"
	apiUsersURL     = "https://kitsu.io/api/edge/users"
)

func getUserLibraryURL(id string) string {
	return fmt.Sprintf("https://kitsu.io/api/edge/users/%s/library-entries", id)
}

func getProfileURL(name string) string {
	return fmt.Sprintf("%s/%s", apiUsersURL, name)
}

func getAnimeURL(slug string) string {
	return fmt.Sprintf("https://kitsu.io/anime/%s", slug)
}
