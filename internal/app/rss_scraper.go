package app

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/Jaeiya/koshime/internal/utils"
	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
)

type RSSDomain int

const (
	Nyaa = RSSDomain(iota)
	AnimeTosho
)

type RSSDomainData struct {
	mirrors       []string
	queryFragment string
}

var domainMap = map[RSSDomain]RSSDomainData{
	Nyaa: {
		[]string{
			"nyaa.si",
			"nyaa.land",
		},
		"/?page=rss&f=0&c=1_2&q=$q",
	},
	AnimeTosho: {
		[]string{"feed.animetosho.org"},
		"/rss2?only_tor=1&q=$q",
	},
}

type RSSEntry struct {
	Title   string
	Date    time.Time
	Torrent string
	Seeds   string
	Size    string
}

type RSSResult struct {
	Entries []RSSEntry
	FeedURL string
	Host    string
}

type RSS struct{}

// FindAnimeFansub uses predefined anime RSS domains to look up the
// specified query and return any results found.
func (rss RSS) FindAnimeFansub(d RSSDomain, query string) (RSSResult, error) {
	var domain RSSDomainData
	if d, ok := domainMap[d]; ok {
		domain = d
	}

	var httpUtil utils.Http
	var parsedURL *url.URL
	var err error

	for _, mirror := range domain.mirrors {
		parsedURL, err = url.Parse(
			fmt.Sprintf(
				"https://%s%s",
				mirror,
				strings.Replace(domain.queryFragment, "$q", url.QueryEscape(query), 1),
			),
		)
		if err != nil {
			return RSSResult{}, err
		}

		req, err := http.NewRequest(http.MethodGet, parsedURL.String(), nil)
		if err != nil {
			return RSSResult{}, err
		}

		resp, err := httpUtil.Do(req, "application/xml", "")
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			return RSSResult{}, err
		}

		// Assume site is temporarily unavailable
		if resp.StatusCode >= 500 {
			continue
		}

		entries, err := rss.parseByHost(parsedURL.Host, string(resp.Body))
		if err != nil {
			return RSSResult{}, err
		}
		return RSSResult{entries, parsedURL.String(), parsedURL.Host}, nil
	}

	return RSSResult{}, fmt.Errorf("all domain mirrors are down")
}

func (rss RSS) parseByHost(host string, xml string) ([]RSSEntry, error) {
	switch {
	case slices.Contains(domainMap[Nyaa].mirrors, host):
		return rss.parseNyaa(xml)

	case slices.Contains(domainMap[AnimeTosho].mirrors, host):
		return rss.parseAnimeTosho(xml)
	}

	return nil, fmt.Errorf("host not supported")
}

func (RSS) parseAnimeTosho(xml string) ([]RSSEntry, error) {
	doc, err := xmlquery.Parse(strings.NewReader(xml))
	if err != nil {
		return []RSSEntry{}, err
	}

	expr, err := xpath.Compile("count(//item)")
	if err != nil {
		return []RSSEntry{}, err
	}
	count, ok := expr.Evaluate(xmlquery.CreateXPathNavigator(doc)).(float64)
	if !ok {
		return []RSSEntry{}, fmt.Errorf("could not eval node to float64")
	}

	results := make([]RSSEntry, int(count))

	for i, d := range xmlquery.Find(doc, "//item") {
		date := d.SelectElement("pubDate").InnerText()
		parsedTime, err := time.Parse(time.RFC1123Z, date)
		if err != nil {
			return nil, err
		}
		results[i] = RSSEntry{
			Title:   d.SelectElement("title").InnerText(),
			Date:    parsedTime,
			Torrent: d.SelectElement("enclosure").SelectAttr("url"),
		}
	}
	return results, nil
}

func (RSS) parseNyaa(xml string) ([]RSSEntry, error) {
	doc, err := xmlquery.Parse(strings.NewReader(xml))
	if err != nil {
		return nil, err
	}

	expr, err := xpath.Compile("count(//item)")
	if err != nil {
		return []RSSEntry{}, err
	}
	count, ok := expr.Evaluate(xmlquery.CreateXPathNavigator(doc)).(float64)
	if !ok {
		return []RSSEntry{}, fmt.Errorf("could not eval node to float64")
	}

	results := make([]RSSEntry, int(count))

	for i, d := range xmlquery.Find(doc, "//item") {
		date := d.SelectElement("pubDate").InnerText()
		parsedTime, err := time.Parse(time.RFC1123Z, date)
		if err != nil {
			return nil, err
		}
		results[i] = RSSEntry{
			Title:   d.SelectElement("title").InnerText(),
			Date:    parsedTime,
			Torrent: d.SelectElement("link").InnerText(),
			Seeds:   d.SelectElement("nyaa:seeders").InnerText(),
			Size:    d.SelectElement("nyaa:size").InnerText(),
		}
	}
	return results, nil
}
