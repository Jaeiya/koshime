package lib

import (
	"fmt"
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
	"ADN":      "AnimeDigitalNetwork",
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

	"DVD":    "DVD",
	"DVDRIP": "DVD",
	"DVDRip": "DVD",
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
	if !strings.Contains(fileName, " ") && !strings.Contains(fileName, ".") {
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
		encoding.WriteString(getFansubEncoding(t))
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

var tmap = map[string]string{
	"360p": "360p",
	"DDP":  "DolbyDigital+",
}

func getFansubEncoding(s string) string {
	keys := []string{
		// Resolutions
		"240p", "360p", "480p", "540p", "720p", "1080p", "1440p",

		// Video Codecs
		"HEVC", "AVC", "AV1", "Hybrid", "Xvid", "XVID", "XviD",
		"H.264", "H.265", "h.265", "h.264", "x265", "x264",

		// Audio Codecs
		"AAC", "AAC2.0", "E-AC-3", "EAC3", "E-AC3", "AC-3",
		"FLAC", "DDP", "TrueHD", "2.0", "Opus", "MP3", "Mp3",

		// Bit Depths
		"10bit", "10-bit", "10-Bit", "8-Bit", "8bit", "8-bit",
	}

	for _, key := range keys {
		switch s {
		case "DDP":
			return "DolbyDigital+ "
		case "E-AC-3", "E-AC3":
			return "EAC3 "
		case "AVC", "x264", "h.264":
			return "H.264 "
		case "H.265", "h.265", "x265":
			return "HEVC "
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
		return source + " "
	}
	return ""
}

func getFansubEpisode(s string, index int, tokens []string) (season string, episode string) {
	if len(s) == 0 {
		return
	}

	if s == "Season" && tokens[index+2][0] == '-' {
		_, err1 := strconv.ParseInt(tokens[index+1], 10, 32)
		_, err2 := strconv.ParseInt(tokens[index+3], 10, 32)
		if err1 == nil && err2 == nil {
			return tokens[index+1], tokens[index+3]
		}
	}

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
// getTokenSeparator returns the most prevalent token
// separator within the given string.
func getTokenSeparator(s string) string {
	spaceCount := strings.Count(s, " ")
	ellipsesCount := strings.Count(s, ".")

	if spaceCount > ellipsesCount {
		return " "
	}

	return "."
}

func newBracketReplaceMap(replacement string) map[string]string {
	return map[string]string{
		"][": replacement,
		")(": replacement,
	}
}
