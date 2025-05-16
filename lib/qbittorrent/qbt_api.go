package qbittorrent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Jaeiya/koshime/lib/utils"
)

const (
	apiLoginURI     = "/auth/login"
	apiFeedItemsURI = "/rss/items"
	apiAddFeedURI   = "/rss/addFeed"
)

var (
	isLoggedIn     = false
	qbtPort        = "8080"
	errNotLoggedIn = fmt.Errorf("you need to be logged into qbittorrent")
	errConnFailed  = fmt.Errorf("cannot connect to qbittorrent")
	httpUtils      = utils.Http{}
)

func Login(password string, port string) error {
	if port != "" {
		qbtPort = port
	}

	// Make sure we can connect to the qbittorrent service
	req, _ := http.NewRequest("HEAD", buildApiUrl(""), nil)
	resp, err := httpUtils.Do(req, "", "")
	if err != nil {
		return errConnFailed
	}

	resp, err = httpUtils.PostForm(
		buildApiUrl(apiLoginURI),
		url.Values{"username": {"admin"}, "password": {password}},
	)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("could not login: %s", string(resp.Body))
	}

	isLoggedIn = true
	return nil
}

func AddFeed(feedURL string, name string) error {
	if !isLoggedIn {
		return errNotLoggedIn
	}

	resp, err := httpUtils.PostForm(buildApiUrl(apiAddFeedURI), url.Values{
		"url":  {feedURL},
		"path": {name},
	})
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to add feed: %s", string(resp.Body))
	}
	return nil
}

func GetFeeds() ([]RSSFeedItem, error) {
	if !isLoggedIn {
		return nil, errNotLoggedIn
	}

	httpUtils := utils.Http{}
	req, _ := http.NewRequest("GET", buildApiUrl(apiFeedItemsURI), nil)
	resp, err := httpUtils.Do(req, "application/json", "")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to get rss items: %s", string(resp.Body))
	}

	var feeds map[string]RSSFeedData
	if err := json.Unmarshal(resp.Body, &feeds); err != nil {
		panic(err)
	}

	items := make([]RSSFeedItem, 0, len(feeds))
	for name, feed := range feeds {
		items = append(items, RSSFeedItem{name, feed.UID, feed.URL})
	}

	return items, nil
}

func buildApiUrl(uri string) string {
	return fmt.Sprintf("http://localhost:%s/api/v2/%s", qbtPort, uri)
}
