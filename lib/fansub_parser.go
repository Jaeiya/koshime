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

var fansubSourceMap = map[string]string{
	"AMZN":     "Amazon",
	"NF":       "Netflix",
	"VIKI":     "Viki",
	"B-Global": "Bilibili-Global",
	"DSNP":     "Disney+",
	"CR":       "CrunchyRoll",

	"WEB-DL": "WebRip",
	"WEB":    "WebRip",
	"WEBRip": "WebRip",

	"BILI":     "Bilibili",
	"Bstation": "Bilibili",
	"BStation": "Bilibili",

	"BD":    "BluRayDisc",
	"BDRip": "BluRayDisc",
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
	fileName = strings.TrimSuffix(fileName, ext)

	if !strings.Contains(fileName, " ") && !strings.Contains(fileName, ".") {
		return FansubInfo{}, fmt.Errorf("cannot tokenize file name")
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

	fileName = utils.ReplaceAll(fileName, fansubReplaceMap)

	tokens := strings.Split(fileName, " ")
	if len(tokens) <= 1 {
		tokens = strings.Split(fileName, ".")
	}

	tokensNoFansub := tokens
	if fileName[0] == '[' {
		fansubName = fileName[1:strings.Index(fileName, "]")]
		fansubTokenCount := strings.Count(fansubName, " ") + 1
		tokensNoFansub = tokens[fansubTokenCount:]
	}

	for i, t := range tokensNoFansub {
		t = utils.RemoveBrackets(t)
		encoding.WriteString(getFansubEncoding(t))
		source.WriteString(getFansubSource(t))
		// Assume only one valid episode number
		if len(episode) == 0 {
			season, episode = getFansubEpisode(t, i, tokensNoFansub)
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

var tmap = map[string]string{
	"360p": "360p",
	"DDP":  "DolbyDigital+",
}

func getFansubEncoding(s string) string {
	keys := []string{
		// Resolutions
		"360p", "480p", "540p", "720p", "1080p", "1440p",

		// Video Codecs
		"HEVC", "AVC", "AV1", "Hybrid",

		// Audio Codecs
		"AAC", "AAC2.0", "E-AC-3", "EAC3", "E-AC3", "AC-3",
		"FLAC", "DDP", "TrueHD", "2.0", "Opus",

		// Encoding Algorithms
		"H.264", "x265", "x264",

		// Bit Depths
		"10bit", "10-bit", "10-Bit", "8-Bit", "8bit", "8-bit",
	}

	for _, key := range keys {
		switch s {
		case "DDP":
			return "DolbyDigital+ "
		case "E-AC-3", "E-AC3":
			return "EAC3"
		default:
			if key == s {
				return key + " "
			}
		}
	}

	return ""
}

func getFansubSource(s string) string {
	if source, exists := fansubSourceMap[s]; exists {
		return source
	}
	return ""
}

func getFansubEpisode(s string, index int, tokens []string) (season string, episode string) {
	if s[0] == 'S' && (len(s) == 6 || len(s) == 8) {
		_, err1 := strconv.ParseInt(s[1:3], 10, 32)
		_, err2 := strconv.ParseInt(s[4:6], 10, 32)
		if err1 == nil && err2 == nil {
			return s[1:3], s[4:]
		}
	}

	// Try to find raw episode
	if s[0] == '-' {
		_, err := strconv.ParseInt(strings.TrimSuffix(tokens[index+1], "v2"), 10, 32)
		if err == nil {
			return "", tokens[index+1]
		}
	}

	return
}
