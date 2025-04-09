package lib

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
)

var rssClient http.Client

type RSSResult struct {
	Content string
	host    string
}

type RSSEntry struct {
	Title   string
	Date    time.Time
	Torrent string
	Seeds   string
	Size    string
}

type RSS struct{}

func (RSS) Get(link string, query string) (RSSResult, error) {
	parsedURL, err := url.Parse(link)
	if err != nil {
		return RSSResult{}, err
	}

	req, err := http.NewRequest("GET", strings.ReplaceAll(link, "$q", query), nil)
	if err != nil {
		return RSSResult{}, err
	}

	req.Header.Set("User-Agent", "Koshime/0.1")
	req.Header.Set("Accept", "application/xml")
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := rssClient.Do(req)
	if err != nil {
		return RSSResult{}, err
	}

	var body []byte

	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(resp.Body)
		defer reader.Close()
		if err != nil {
			return RSSResult{}, err
		}

		body, err = io.ReadAll(reader)
		if err != nil {
			return RSSResult{}, err
		}

	} else {
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return RSSResult{}, err
		}
	}

	return RSSResult{
		host:    parsedURL.Host,
		Content: string(body),
	}, nil
}

func (RSS) Parse(result RSSResult) []RSSEntry {
	switch result.host {
	case "nyaa.si", "www.nyaa.si":
		return parseNyaa(result)
	}

	panic("host not supported")
}

func parseNyaa(result RSSResult) []RSSEntry {
	doc, err := xmlquery.Parse(strings.NewReader(result.Content))
	if err != nil {
		panic(err)
	}

	expr, err := xpath.Compile("count(//item)")
	count := expr.Evaluate(xmlquery.CreateXPathNavigator(doc)).(float64)

	results := make([]RSSEntry, int(count))

	for i, d := range xmlquery.Find(doc, "//item") {
		date := d.SelectElement("pubDate").InnerText()
		parsedTime, err := time.Parse(time.RFC1123Z, date)
		if err != nil {
			panic(err)
		}
		results[i] = RSSEntry{
			Title:   d.SelectElement("title").InnerText(),
			Date:    parsedTime,
			Torrent: d.SelectElement("link").InnerText(),
			Seeds:   d.SelectElement("nyaa:seeders").InnerText(),
			Size:    d.SelectElement("nyaa:size").InnerText(),
		}
	}
	return results
}
