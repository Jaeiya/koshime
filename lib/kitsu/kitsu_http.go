package kitsu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Jaeiya/koshime/lib/utils"
)

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

func newAPIRequest[T any](options APIReqOptions, data *T) (int, error) {
	method := string(options.method)
	url := string(options.url)
	contentType := string(options.contentType)

	var req *http.Request
	var err error

	switch options.method {
	case apiGet, apiHead, apiDelete:
		req, err = http.NewRequest(method, url, nil)
	case apiPost, apiPatch:
		req, err = http.NewRequest(method, url, bytes.NewBuffer(options.payload))
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

	var httpUtil utils.Http
	resp, err := httpUtil.Do(req, string(vndAPIContent), contentType)
	if err != nil {
		return -1, err
	}

	err = validateResponse(resp)
	if err != nil {
		return resp.StatusCode, err
	}

	if data != nil {
		err = json.NewDecoder(bytes.NewReader(resp.Body)).Decode(data)
		if err != nil {
			return resp.StatusCode, err
		}
	}

	return resp.StatusCode, nil
}

func validateResponse(resp utils.HttpResponse) error {
	body := string(resp.Body)

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.StatusText)
	}

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		var errData APIErrorData
		err := json.Unmarshal(resp.Body, &errData)
		if err == nil {
			if len(errData.Errors) > 0 {
				return fmt.Errorf("%s", errData)
			}
		}

		var authErrData AuthErrorData
		err = json.Unmarshal(resp.Body, &authErrData)
		if err == nil {
			return fmt.Errorf("%s", authErrData)
		}

		return fmt.Errorf(
			"ClientError::HTTP %d: %s\n%s",
			resp.StatusCode,
			resp.StatusText,
			body,
		)
	}

	if resp.StatusCode >= 500 {
		return fmt.Errorf(
			"ServerError::HTTP %d: %s",
			resp.StatusCode,
			resp.StatusText,
		)
	}
	return nil
}
