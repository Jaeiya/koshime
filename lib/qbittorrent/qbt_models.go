package qbittorrent

import (
	"fmt"
	"slices"
)

type RSSFeedData struct {
	UID string `json:"uid"`
	URL string `json:"url"`
}

type RSSFeedItem struct {
	Name string
	UID  string
	URL  string
}

func (i RSSFeedItem) String() string {
	return fmt.Sprintf("Name: %s\n UID: %s\n URL: %s\n", i.Name, i.UID, i.URL)
}

type RSSRulesMap map[string]RSSRule

type RSSRule struct {
	Enabled                   bool     `json:"enabled"`
	MustContain               string   `json:"mustContain"`
	MustNotContain            string   `json:"mustNotContain"`
	UseRegex                  bool     `json:"useRegex"`
	EpisodeFilter             string   `json:"episodeFilter"`
	SmartFilter               bool     `json:"smartFilter"`
	PreviouslyMatchedEpisodes []string `json:"previouslyMatchedEpisodes"`
	AffectedFeeds             []string `json:"affectedFeeds"`
	IgnoreDays                int      `json:"ignoreDays"`
	LastMatch                 string   `json:"lastMatch"`
	AddPaused                 bool     `json:"addPaused"`
	AssignedCategory          string   `json:"assignedCategory"`
	SavePath                  string   `json:"savePath"`
}

func (r *RSSRule) AddFeed(feed string) bool {
	idx := slices.Index(r.AffectedFeeds, feed)
	if idx > -1 {
		return false
	}

	r.AffectedFeeds = append(r.AffectedFeeds, feed)
	return true
}

func (r *RSSRule) RemoveFeed(feed string) bool {
	idx := slices.Index(r.AffectedFeeds, feed)
	if idx < 0 {
		return false
	}

	r.AffectedFeeds = slices.Delete(r.AffectedFeeds, idx, idx+1)
	return true
}
