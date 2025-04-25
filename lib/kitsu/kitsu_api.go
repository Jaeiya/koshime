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
	err = postJSON(apiAuthTokenURL, payload, "", &data)
	if err != nil {
		return AuthToken{}, err
	}

	return data, nil
}

func GetProfile(userName string) (ProfileData, error) {
	profileURL, err := getProfileURL(userName)
	if err != nil {
		return ProfileData{}, err
	}

	var data ProfileData
	err = getJSON(profileURL, &data)
	if err != nil {
		return ProfileData{}, err
	}

	return data, nil
}

func GetProfileLink(userName string) string {
	p, _ := url.JoinPath("https://kitsu.app/users", userName)
	return p
}
