package lib

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/Jaeiya/koshime/lib/utils"
	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
)

type RSSDomain int

const (
	Nyaa = RSSDomain(iota)
)

type RSSDomainData struct {
	mirrors       []string
	queryFragment string
}

var domainMap = map[RSSDomain]RSSDomainData{
	Nyaa: {
		[]string{
			"nyaa.land",
			"nyaa.si",
		},
		"/?page=rss&f=0&c=1_2&q=$q",
	},
}

type RSSResult struct {
	Content string
	Host    string
}

type RSSEntry struct {
	Title   string
	Date    time.Time
	Torrent string
	Seeds   string
	Size    string
}

type RSS struct{}

// Get returns an RSSResult, which is intended to be passed to
// the Parse func. The content will contain the raw XML returned
// from the RSS link.
//
// 🟠 The link must contain the query replacement sequence '$q'
// or an error will be returned. If the service does not have
// the ability to be queried, then it cannot be used in this
// context.
//
//	:: Example Link ::
//	page.com/rss.php?q=$q
//
// Where `$q` will be replaced by the specified query
// parameter.
func (RSS) Get(d RSSDomain, query string) (RSSResult, error) {
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

		req, err := http.NewRequest("GET", parsedURL.String(), nil)
		if err != nil {
			return RSSResult{}, err
		}

		resp, err := httpUtil.Do(req, "application/xml", "")
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			} else {
				return RSSResult{}, err
			}
		}

		// Assume site is temporarily unavailable
		if resp.StatusCode >= 500 {
			continue
		}

		return RSSResult{
			Host:    parsedURL.Host,
			Content: string(resp.Body),
		}, nil
	}

	return RSSResult{}, fmt.Errorf("all domain mirrors are down")
}

func (rss RSS) Parse(result RSSResult) ([]RSSEntry, error) {
	switch {
	case slices.Contains(domainMap[Nyaa].mirrors, result.Host):
		return rss.parseNyaa(result), nil
	}

	return []RSSEntry{}, fmt.Errorf("host not supported")
}

func (RSS) parseNyaa(result RSSResult) []RSSEntry {
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
