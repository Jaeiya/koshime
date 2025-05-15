package qbittorrent

import "fmt"

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
