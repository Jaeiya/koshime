package utils

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

const userAgent = "Koshime/0.1"

var client = func() http.Client {
	jar, _ := cookiejar.New(nil)
	return http.Client{
		Timeout: 5 * time.Second,
		Jar:     jar,
	}
}()

type HttpResponse struct {
	StatusCode int
	StatusText string
	Body       []byte
	Cookies    []*http.Cookie
}

type Http struct{}

func (Http) GetUserAgent() string {
	return userAgent
}

// Do executes an http request with the specified Accept
// and Content-Type headers, which can be left empty.
//
// 🟡 If accept is empty, it will use text/plain
// 🟡 If contentType is empty, it will not be used
// 🟡 All cookies will be saved and sent with subsequent requests
func (h Http) Do(req *http.Request, accept, contentType string) (HttpResponse, error) {
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if accept == "" {
		accept = "text/plain"
	}

	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept-Encoding", "gzip")

	//nolint:gosec // all links urls are internal
	resp, err := client.Do(req)
	if err != nil {
		return HttpResponse{}, err
	}
	defer resp.Body.Close()

	body, err := h.ReadResponseBody(resp)
	if err != nil {
		return HttpResponse{}, err
	}

	return HttpResponse{
		StatusCode: resp.StatusCode,
		StatusText: http.StatusText(resp.StatusCode),
		Body:       body,
	}, nil
}

func (h Http) PostForm(url string, values url.Values) (HttpResponse, error) {
	resp, err := client.PostForm(url, values)
	if err != nil {
		return HttpResponse{}, err
	}
	defer resp.Body.Close()

	body, err := h.ReadResponseBody(resp)
	if err != nil {
		return HttpResponse{}, err
	}

	return HttpResponse{
		StatusCode: resp.StatusCode,
		StatusText: http.StatusText(resp.StatusCode),
		Body:       body,
		Cookies:    resp.Cookies(),
	}, nil
}

func (Http) ReadResponseBody(resp *http.Response) ([]byte, error) {
	defer func() { _ = resp.Body.Close() }()

	reader := io.Reader(resp.Body)

	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer func() { _ = gzReader.Close() }()
		reader = gzReader
	}

	return io.ReadAll(reader)
}
