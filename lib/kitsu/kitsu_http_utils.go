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

const clientAgent = "Koshime/0.1"

var client = &http.Client{}

type KitsuMethod string

const (
	apiGet    = KitsuMethod("GET")
	apiPost   = KitsuMethod("POST")
	apiPatch  = KitsuMethod("PATCH")
	apiHead   = KitsuMethod("HEAD")
	apiDelete = KitsuMethod("DELETE")
)

type APIReqOptions struct {
	method      KitsuMethod
	url         APIUrl
	contentType kitsuContentType
	payload     []byte
	token       string
}

type kitsuContentType string

const (
	vndAPIContent = kitsuContentType("application/vnd.api+json")
	jsonContent   = kitsuContentType("application/json")
)

// TODO  Use this to create better HTTP error messages
// type BadRequestResp struct {
// 	Errors []struct {
// 		Title  string
// 		Detail string
// 		Code   string
// 		Status string
// 	}
// }

func newAPIRequest[T any](options APIReqOptions, data *T) (int, error) {
	var req *http.Request
	var err error

	switch options.method {
	case apiGet, apiHead, apiDelete:
		req, err = newKitsuRequest(
			string(options.method),
			string(options.url),
			options.contentType,
			nil,
		)
	case apiPost, apiPatch:
		req, err = newKitsuRequest(
			string(options.method),
			string(options.url),
			options.contentType,
			bytes.NewBuffer(options.payload),
		)
	}
	if err != nil {
		return -1, err
	}

	if req == nil {
		return -1, fmt.Errorf("invalid kitsu http method")
	}

	if options.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.token))
	}

	resp, err := client.Do(req)
	if err != nil {
		return -1, err
	}

	body, err := readResponseBody(resp)
	if err != nil {
		return resp.StatusCode, err
	}

	err = validateResponse(resp, string(body))
	if err != nil {
		return resp.StatusCode, err
	}

	if data != nil {
		err = json.NewDecoder(bytes.NewReader(body)).Decode(data)
		if err != nil {
			return resp.StatusCode, err
		}
	}

	return resp.StatusCode, nil
}

func newKitsuRequest(
	method, url string,
	ct kitsuContentType,
	body io.Reader,
) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", string(ct))
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

func validateResponse(resp *http.Response, body string) error {
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		if strings.Contains(body, "invalid_grant") {
			return fmt.Errorf("invalid username or password")
		}
		return fmt.Errorf(
			"ClientError::HTTP %d: %s\n%s",
			resp.StatusCode,
			http.StatusText(resp.StatusCode),
			body,
		)
	}

	if resp.StatusCode >= 500 {
		return fmt.Errorf(
			"ServerError::HTTP %d: %s",
			resp.StatusCode,
			http.StatusText(resp.StatusCode),
		)
	}
	return nil
}
