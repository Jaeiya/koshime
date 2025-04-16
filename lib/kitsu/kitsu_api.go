package kitsu

import (
	"encoding/json"

	"github.com/jaeiya/koshime/lib/utils"
)

type TokenData struct {
	Token        string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	CreatedAt    int    `json:"created_at"`
}

func GetAuthToken(userName, password string) (TokenData, error) {
	credentials := map[string]string{
		"grant_type": "password",
		"username":   userName,
		"password":   password,
	}

	payload, err := json.Marshal(credentials)
	if err != nil {
		return TokenData{}, err
	}

	var data TokenData
	err = utils.PostJSON(apiAuthTokenURL, payload, &data)
	if err != nil {
		panic(err)
	}

	return data, nil
}
