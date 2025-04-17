package utils

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const userAgent = "Koshime/0.1"

var client = &http.Client{}

func PostJSON[T any](url string, content []byte, data *T) error {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(content))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var reader io.ReadCloser
	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return err
		}
	} else {
		reader = resp.Body
	if resp.Header.Get("Content-Type") == "text/html" {
		return fmt.Errorf("HTTP %d: expected JSON response but got HTML", resp.StatusCode)
	}
	defer reader.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(reader)
		bodyStr := string(body)
		if resp.StatusCode == 400 {
			if strings.Contains(bodyStr, "invalid_grant") {
				return fmt.Errorf("invalid username or password")
			}
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, bodyStr)
	}

	err = json.NewDecoder(reader).Decode(data)
	if err != nil {
		return err
	}

	return nil
}
