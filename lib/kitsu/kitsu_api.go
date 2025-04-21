package kitsu

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const clientAgent = "Koshime/0.1"

var client = &http.Client{}

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
	p, _ := url.JoinPath(usersURL, userName)
	return p
}

func getJSON[T any](url string, data *T) error {
	req, err := NewJsonRequest("GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}

	bodyStr := string(body)

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return fmt.Errorf(
			"ClientError::HTTP %d: %s\n%s",
			resp.StatusCode,
			http.StatusText(resp.StatusCode),
			bodyStr,
		)
	}

	if resp.StatusCode >= 500 {
		return fmt.Errorf(
			"ServerError::HTTP %d: %s",
			resp.StatusCode,
			http.StatusText(resp.StatusCode),
		)
	}

	err = json.NewDecoder(bytes.NewReader(body)).Decode(data)
	if err != nil {
		return err
	}

	return nil
}

func postJSON[T any](url string, payload []byte, bearerToken string, data *T) error {
	req, err := NewJsonRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	if bearerToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bearerToken))
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	bodyStr := string(body)

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		if strings.Contains(bodyStr, "invalid_grant") {
			return fmt.Errorf("invalid username or password")
		}
		return fmt.Errorf(
			"ClientError::HTTP %d: %s\n%s",
			resp.StatusCode,
			http.StatusText(resp.StatusCode),
			bodyStr,
		)
	}

	if resp.StatusCode >= 500 {
		return fmt.Errorf(
			"ServerError::HTTP %d: %s",
			resp.StatusCode,
			http.StatusText(resp.StatusCode),
		)
	}

	err = json.NewDecoder(bytes.NewReader(body)).Decode(data)
	if err != nil {
		return err
	}

	return nil
}

func NewJsonRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("User-Agent", clientAgent)
	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("Accept-Encoding", "gzip")

	return req, nil
}

func readResponseBody(resp *http.Response) ([]byte, error) {
	var reader io.ReadCloser
	var err error
	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			resp.Body.Close()
			return []byte{}, err
		}
		defer func() {
			reader.Close()
			resp.Body.Close()
		}()
	} else {
		reader = resp.Body
		defer reader.Close()
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}
