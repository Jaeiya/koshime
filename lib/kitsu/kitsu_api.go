package kitsu

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/Jaeiya/koshime/lib/database"
)

func GetAuthToken(userName, password string) (AuthToken, error) {
	credentials := map[string]string{
		"grant_type": "password",
		"username":   userName,
		"password":   password,
	}

	payload, err := json.Marshal(credentials)
	if err != nil {
		return AuthToken{}, err
	}

	var data AuthToken
	opts := APIReqOptions{
		method:      apiPost,
		url:         apiAuthTokenURL,
		payload:     payload,
		contentType: jsonContent,
	}
	if _, err = newAPIRequest(opts, &data); err != nil {
		return AuthToken{}, err
	}

	return data, nil
}

func GetLibraryAnime(userID string, status LibAnimeStatus) ([]database.LibraryEntry, error) {
	qurl, err := getUserLibAnimeQURL(userID, status)
	if err != nil {
		return nil, err
	}

	var respData LibraryAnime
	opt := APIReqOptions{
		method:      apiGet,
		url:         qurl,
		contentType: vndAPIContent,
	}
	if _, err = newAPIRequest(opt, &respData); err != nil {
		return nil, err
	}

	entries := make([]database.LibraryEntry, len(respData.Data))
	for i, item := range respData.Data {
		anime := respData.Included[i]
		entries[i] = database.LibraryEntry{
			ID:        respData.Included[i].ID,
			LibID:     item.LibID,
			JPN_Title: anime.Attributes.Titles.Romaji,
			ENG_Title: anime.Attributes.Titles.English,
			AltTitles: anime.Attributes.AltTitles,
			Episodes:  anime.Attributes.EpCount,
			Progress:  item.Attributes.Progress,
			Slug:      anime.Attributes.Slug,
		}
	}

	return entries, nil
}

// AddAnime adds an anime to the users Kitsu library with
// the specified status. On success, the library ID of
// the added anime is returned.
func AddAnime(animeID, userID, token string, status LibAnimeStatus) (string, error) {
	respData := struct {
		Data struct {
			LibID string `json:"id"`
		}
	}{}

	payload, err := newAddAnimePayload(animeID, userID, status)
	if err != nil {
		return "", err
	}

	opts := APIReqOptions{
		method:      apiPost,
		url:         apiLibraryURL,
		contentType: vndAPIContent,
		payload:     payload,
		token:       token,
	}
	if _, err = newAPIRequest(opts, &respData); err != nil {
		return "", err
	}
	return respData.Data.LibID, nil
}

// SetLibAnimeStatus sets the status of a specific anime within
// the users library. On success, the active status is returned.
func SetLibAnimeStatus(libID, token string, status LibAnimeStatus) (LibAnimeStatus, error) {
	payload, err := newAnimeStatusPayload(libID, status)
	if err != nil {
		return "", err
	}

	respData := struct {
		Data struct {
			Id         string
			Attributes struct {
				Status string
			}
		}
	}{}

	opts := APIReqOptions{
		method:      apiPatch,
		url:         getLibEntryURL(libID),
		contentType: vndAPIContent,
		payload:     payload,
		token:       token,
	}
	if _, err = newAPIRequest(opts, &respData); err != nil {
		return "", err
	}

	return LibAnimeStatus(respData.Data.Attributes.Status), nil
}

// DeleteLibAnime deletes a specific anime from a users
// library. On success, it should return a 204 HTTP
// status code.
func DeleteLibAnime(libID, token string) (int, error) {
	opts := APIReqOptions{
		method:      apiDelete,
		url:         getLibEntryURL(libID),
		contentType: vndAPIContent,
		token:       token,
	}
	var none *string
	status, err := newAPIRequest(opts, none)
	if err != nil {
		return status, err
	}
	return status, nil
}

func GetProfile(userName string) (database.Profile, error) {
	qurl, err := getProfileQURL(userName)
	if err != nil {
		return database.Profile{}, err
	}

	var respData ProfileData
	_, err = newAPIRequest(APIReqOptions{
		method:      apiGet,
		url:         qurl,
		contentType: vndAPIContent,
	}, &respData)
	if err != nil {
		return database.Profile{}, err
	}

	if len(respData.Data) == 0 {
		return database.Profile{}, fmt.Errorf("profile not found")
	}

	profileData := respData.Data[0]
	profileStats := respData.Included[0]

	profile := database.Profile{
		ID:              profileData.ID,
		Username:        profileData.Attributes.Name,
		About:           profileData.Attributes.About,
		Birthday:        profileData.Attributes.Birthday,
		Location:        profileData.Attributes.Location,
		Gender:          profileData.Attributes.Gender,
		CreatedAt:       profileData.Attributes.CreatedAt,
		SecondsWatched:  profileStats.Attributes.Stats.SecondsWatched,
		CompletedSeries: profileStats.Attributes.Stats.CompletedAnime,
		LastUpdateSec:   time.Now().Unix(),
	}

	return profile, nil
}

func GetProfileLink(userName string) string {
	p, _ := url.JoinPath(KitsuDomain+"/users", userName)
	return p
}
