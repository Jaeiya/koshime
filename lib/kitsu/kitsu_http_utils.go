package kitsu

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func newJSONRequest(method, url string, body io.Reader) (*http.Request, error) {
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

func getJSON[T any](url APIUrl, data *T) error {
	req, err := newJSONRequest("GET", string(url), nil)
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

func postJSON[T any](url APIUrl, payload []byte, bearerToken string, data *T) error {
	req, err := newJSONRequest("POST", string(url), bytes.NewBuffer(payload))
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
