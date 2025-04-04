package lib

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jaeiya/koshime/lib/utils"
)

var fansubReplaceMap = map[string]string{
	"][": " ",
	")(": " ",
}

var fansubEncodingMap = map[string]string{
	"1920x1080": "1080p",
	"1280x720":  "720p",

	"av1":        "AV1",
	"aac":        "AAC",
	"aac2.0":     "AAC2.0",
	"hevc-10bit": "HEVC 10-bit",
	"hevc":       "HEVC",
	"h.265":      "HEVC",
	"x265":       "HEVC",
	"avc":        "H.264",
	"h.264":      "H.264",
	"x264":       "H.264",

	"ddp":    "DolbyDigital+",
	"ddp2.0": "DolbyDigital+2.0",
	"truehd": "TrueHD",
	"opus":   "Opus",
	"mp3":    "Mp3",
	"flac":   "FLAC",
	"e-ac-3": "EAC3",
	"e-ac3":  "EAC3",
	"eac3":   "EAC3",

	"10bit":  "10-bit",
	"10-bit": "10-bit",

	"8bit":  "8-bit",
	"8-bit": "8-bit",
}

var fansubSourceMap = map[string]string{
	"amzn":     "Amazon",
	"nf":       "Netflix",
	"viki":     "Viki",
	"adn":      "AnimeDigitalNetwork",
	"baha":     "BahamutAnimeMadness",
	"b-global": "Bilibili-Global",
	"dsnp":     "Disney+",
	"cr":       "CrunchyRoll",
	"hidive":   "HIDIVE",
	"yt":       "YouTube",
	"cx":       "TV Asahi",

	"bs8":  "TV Tokyo",
	"bs11": "TV Tokyo",

	"web-dl": "WebRip",
	"webrip": "WebRip",
	"WEB":    "WebRip",

	"bili":     "Bilibili",
	"bstation": "Bilibili",

	"bd":    "BluRayDisc",
	"bdrip": "BluRayDisc",

	"dvd":    "DVD",
	"dvdrip": "DVD",
}

var fansubExtMap = map[string]struct{}{
	".mp4": {},
	".mkv": {},
	".avi": {},
	".mpg": {},
	".wmv": {},
}

type FansubInfo struct {
	Fansub   string
	Title    string
	Encoding string
	Season   string
	Episode  string
	Source   string
}

type Fansub struct{}

func (Fansub) Parse(fileName string) (FansubInfo, error) {
	ext := filepath.Ext(fileName)
	if _, hasExt := fansubExtMap[ext]; hasExt {
		fileName = strings.TrimSuffix(fileName, ext)
	}

	if !strings.Contains(fileName, " ") && !strings.Contains(fileName, ".") &&
		!strings.Contains(fileName, "_") {
		return FansubInfo{}, fmt.Errorf("unsupported file name")
	}

	if i := strings.Index(strings.ToLower(fileName), "batch"); i > 0 {
		leadingChar := fileName[i-1]
		if leadingChar == '[' || leadingChar == '(' {
			return FansubInfo{}, fmt.Errorf("batch files not supported")
		}
	}

	var fansubName string
	var encoding strings.Builder
	var source strings.Builder
	var title strings.Builder
	var episode string
	var season string

	if fileName[0] == '[' {
		fansubEndIndex := strings.Index(fileName, "]")
		fansubName = fileName[1:strings.Index(fileName, "]")]
		fileName = fileName[fansubEndIndex+1:]
	}

	separator := getTokenSeparator(fileName)
	fileName = utils.ReplaceAll(fileName, newBracketReplaceMap(separator))
	fileName = strings.TrimPrefix(fileName, separator)
	tokens := strings.Split(fileName, separator)

	for i, t := range tokens {
		t = utils.RemoveBrackets(t)
		encoding.WriteString(getFansubEncoding(t, i, tokens))
		source.WriteString(getFansubSource(t))
		// Assume only one valid episode number
		if len(episode) == 0 {
			season, episode = getFansubEpisode(t, i, tokens)
		}

		// Title can be anything but should not be found after meta-data
		foundMetaData := encoding.Len() > 0 || source.Len() > 0 || len(episode) > 0
		if !foundMetaData {
			title.WriteString(t + " ")
		}
	}

	info := FansubInfo{}

	info.Fansub = fansubName
	info.Episode = episode
	info.Season = season
	info.Title = strings.TrimRight(title.String(), "- ~")
	info.Encoding = strings.TrimSpace(encoding.String())
	info.Source = strings.TrimSpace(source.String())
	return info, nil
}

func getFansubEncoding(s string, index int, tokens []string) string {
	resolutions := []string{
		"240p", "360p", "480p", "540p", "720p", "1080p", "1440p", "2160p",
	}

	if val, exists := fansubEncodingMap[strings.ToLower(s)]; exists {
		// Catch "AAC 2.0" edge case
		if val == "AAC" && index+1 < len(tokens) {
			if utils.RemoveBrackets(tokens[index+1]) == "2.0" {
				return val + "2.0 "
			}
		}
		return val + " "
	}

	// Catch "H 264" edge case
	if s == "264" {
		lastToken := utils.RemoveBrackets(strings.ToLower(tokens[index-1]))
		if lastToken == "h" {
			return "H.264 "
		}
	}

	for _, res := range resolutions {
		if res == s {
			return res + " "
		}
	}

	return ""
}

func getFansubSource(s string) string {
	if source, exists := fansubSourceMap[strings.ToLower(s)]; exists {
		return source + " "
	}
	return ""
}

func getFansubEpisode(s string, index int, tokens []string) (season string, episode string) {
	if len(s) == 0 {
		return
	}

	trimEpVersion := func(episode string) string {
		return strings.TrimSuffix(episode, "v2")
	}

	// Catch "Season # - #" for season & episode
	if s == "Season" && tokens[index+2][0] == '-' {
		_, err1 := strconv.ParseInt(tokens[index+1], 10, 32)
		_, err2 := strconv.ParseInt(tokens[index+3], 10, 32)
		if err1 == nil && err2 == nil {
			return tokens[index+1], tokens[index+3]
		}
	}

	// Catch "S##E##" for season & episode
	if s[0] == 'S' && (len(s) == 6 || len(s) == 8) {
		_, err1 := strconv.ParseInt(s[1:3], 10, 32)
		episode := trimEpVersion(s[4:])
		_, err2 := strconv.ParseInt(episode, 10, 32)
		if err1 == nil && err2 == nil {
			return s[1:3], episode
		}
	}

	// Catch "S# - #" for season & episode
	if s[0] == 'S' && (len(s) == 2 || len(s) == 3) {
		_, err1 := strconv.ParseInt(s[1:], 10, 32)
		episode := trimEpVersion(tokens[index+2])
		_, err2 := strconv.ParseInt(episode, 10, 32)
		if err1 == nil && err2 == nil {
			return s[1:], tokens[index+2]
		}

	}

	// Catch "- #" for episode
	if s[0] == '-' {
		episode := trimEpVersion(tokens[index+1])
		_, err := strconv.ParseInt(episode, 10, 32)
		if err == nil {
			return "", tokens[index+1]
		}
	}

	// Catch "EP#" for episode
	if len(s) > 2 && s[:2] == "EP" {
		episode := trimEpVersion(s[2:])
		_, err := strconv.ParseInt(episode, 10, 32)
		if err == nil {
			return "", episode
		}
	}

	return
}

// getTokenSeparator returns the most prevalent token
// separator within the given string.
func getTokenSeparator(s string) string {
	spaceCount := strings.Count(s, " ")
	ellipsesCount := strings.Count(s, ".")
	underscoreCount := strings.Count(s, "_")

	if spaceCount > ellipsesCount && spaceCount > underscoreCount {
		return " "
	}

	if ellipsesCount > underscoreCount {
		return "."
	}

	return "_"
}

func newBracketReplaceMap(replacement string) map[string]string {
	return map[string]string{
		"][": replacement,
		")(": replacement,
	}
}
