package kitsu

import (
	"encoding/json"
	"net/url"
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

func GetProfile(userName string) (ProfileData, error) {
	qurl, err := getProfileQURL(userName)
	if err != nil {
		return ProfileData{}, err
	}

	var data ProfileData
	_, err = newAPIRequest(APIReqOptions{
		method:      apiGet,
		url:         qurl.ToAPIUrl(),
		contentType: vndAPIContent,
	}, &data)
	if err != nil {
		return ProfileData{}, err
	}

	return data, nil
}

func GetProfileLink(userName string) string {
	p, _ := url.JoinPath("https://kitsu.app/users", userName)
	return p
}
