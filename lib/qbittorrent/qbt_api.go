package qbittorrent

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"

	"github.com/Jaeiya/koshime/lib/utils"
)

type RuleMode int8

const (
	CreateRule = RuleMode(iota)
	ModifyRule
)

type ApiUri string

const (
	// Auth
	apiLoginURI  = ApiUri("auth/login")
	apiLogoutURI = ApiUri("auth/logout")

	// RSS
	apiRulesURI     = ApiUri("rss/rules")
	apiSaveRuleURI  = ApiUri("rss/setRule")
	apiFeedItemsURI = ApiUri("rss/items")
	apiAddFeedURI   = ApiUri("rss/addFeed")
)

var (
	qbtPort        = "8080"
	errNotLoggedIn = fmt.Errorf("you need to be logged into qbittorrent")
	errConnFailed  = fmt.Errorf("cannot connect to qbittorrent")
	httpUtils      = utils.Http{}
	ruleCache      = map[string]RSSRule{}
)

type qBittorrentAPI struct {
	Port string
}

func NewLogin(password string, port string) (*qBittorrentAPI, error) {
	if port != "" {
		qbtPort = port
	}

	// Make sure we can connect to the qbittorrent service
	req, _ := http.NewRequest("HEAD", buildApiUrl(""), nil)
	resp, err := httpUtils.Do(req, "", "")
	if err != nil {
		return nil, errConnFailed
	}

	resp, err = httpUtils.PostForm(
		buildApiUrl(apiLoginURI),
		url.Values{"username": {"admin"}, "password": {password}},
	)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("could not login: %s", string(resp.Body))
	}

	if len(resp.Cookies) == 0 {
		return nil, fmt.Errorf("missing session ID cookie")
	}

	qb := &qBittorrentAPI{Port: port}

	rules, err := qb.Rules()
	if err != nil {
		return nil, fmt.Errorf("failed to cache rules: %w", err)
	}

	maps.Copy(ruleCache, rules)

	return qb, nil
}

func (qb qBittorrentAPI) AddFeed(name, feedURL string) error {
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

func (qb qBittorrentAPI) Feeds() ([]RSSFeedItem, error) {
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

func (qb qBittorrentAPI) AddRule(name string, rule RSSRule) error {
	err := qb.saveRule(name, rule, CreateRule)
	if err != nil {
		return fmt.Errorf("failed to add rule: %w", err)
	}
	ruleCache[name] = rule
	return nil
}

func (qb qBittorrentAPI) AddRuleFeed(name, feed string) error {
	rule, exists := ruleCache[name]
	if !exists {
		return fmt.Errorf("rule does not exist: [%s]", name)
	}

	rule.AddFeed(feed)

	err := qb.saveRule(name, rule, ModifyRule)
	if err != nil {
		return fmt.Errorf("failed to add rule feed: %w", err)
	}

	ruleCache[name] = rule
	return nil
}

func (qb qBittorrentAPI) DeleteRuleFeed(name, feed string) error {
	rule, exists := ruleCache[name]
	if !exists {
		return fmt.Errorf("rule does not exist: [%s]", name)
	}

	rule.RemoveFeed(feed)
	err := qb.saveRule(name, rule, ModifyRule)
	if err != nil {
		return fmt.Errorf("failed to add rule feed: %w", err)
	}

	ruleCache[name] = rule
	return nil
}

func (qb qBittorrentAPI) Rules() (RSSRulesMap, error) {
	req, _ := http.NewRequest("GET", buildApiUrl(apiRulesURI), nil)
	resp, err := httpUtils.Do(req, "application/json", "")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to get rules: %s", string(resp.Body))
	}

	var rules RSSRulesMap
	if err := json.Unmarshal(resp.Body, &rules); err != nil {
		return nil, fmt.Errorf("failed to parse rules: %w", err)
	}

	return rules, nil
}

func (qb qBittorrentAPI) saveRule(name string, rule RSSRule, mode RuleMode) error {
	_, ruleExists := ruleCache[name]

	switch mode {
	case CreateRule:
		if ruleExists {
			return fmt.Errorf("rule already exists")
		}

	case ModifyRule:
		if !ruleExists {
			return fmt.Errorf("cannot modify non-existent rule: [%s]", name)
		}

	default:
		return fmt.Errorf("unsupported rule mode: %d", mode)
	}

	jsonRule, err := json.Marshal(rule)
	if err != nil {
		return fmt.Errorf("failed to parse rule as json: %w", err)
	}

	resp, err := httpUtils.PostForm(buildApiUrl(apiSaveRuleURI), url.Values{
		"ruleName": {name},
		"ruleDef":  {string(jsonRule)},
	})
	if err != nil {
		return fmt.Errorf("failed to save rule: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to save rule: %s", string(resp.Body))
	}

	return nil
}

func (qb qBittorrentAPI) Logout() error {
	req, _ := http.NewRequest("POST", buildApiUrl(apiLogoutURI), nil)
	resp, err := httpUtils.Do(req, "", "")
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to logout: %s", string(resp.Body))
	}

	return nil
}

func buildApiUrl(uri ApiUri) string {
	return fmt.Sprintf("http://localhost:%s/api/v2/%s", qbtPort, uri)
}
