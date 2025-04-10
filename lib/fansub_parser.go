package lib

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jaeiya/koshime/lib/utils"
)

var fansubReplaceMap = map[string]string{
	"][": " ",
	")(": " ",
}

var fansubEncodingMap = map[string]string{
	"1920x1080":  "1080p",
	"1920x1080i": "1080i",
	"1440x1080":  "1080p-Squeezed",
	"1280x720":   "720p",
	"854x480":    "480p",
	"640x360":    "360p",

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

	"hdr": "HDR",
	"sdr": "SDR",

	"ddp":    "DolbyDigital+",
	"ddp2.0": "DolbyDigital+2.0",
	"ddp5.1": "DolbyDigital+5.1",
	"dv":     "DolbyVision",
	"atmos":  "DolbyAtmos",

	"truehd":    "TrueHD",
	"truehd7.1": "TrueHD7.1",
	"opus":      "Opus",
	"mp3":       "Mp3",
	"flac":      "FLAC",
	"e-ac-3":    "EAC3",
	"e-ac3":     "EAC3",
	"eac3":      "EAC3",
	"eac-3":     "EAC3",

	"dual-audio": "DualAudio",

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

	"tx":    "Japan TV Broadcast",
	"tbs":   "Japan TV Broadcast",
	"bs8":   "Japan TV Broadcast",
	"bs11":  "Japan TV Broadcast",
	"bs260": "Japan TV Broadcast",
	"at-x":  "Japan TV Broadcast",

	"web-dl": "Web-Download",

	"webrip": "WebRip",
	"web":    "WebRip",

	"bili":     "Bilibili",
	"bstation": "Bilibili",

	"bluray": "BlueRayDisc",
	"bd":     "BluRayDisc",
	"bdrip":  "BluRayDisc",

	"dvd":    "DVD",
	"dvdrip": "DVD",
}

var fansubExtMap = map[string]struct{}{
	".mp4": {},
	".mkv": {},
	".avi": {},
	".mpg": {},
	".wmv": {},
	".ts":  {},
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

	if !strings.ContainsAny(fileName, " ._") {
		return FansubInfo{}, fmt.Errorf("unsupported file name")
	}

	var fansubName, episode, season string
	var encoding, source, title strings.Builder

	tokens, err := getFansubTokens(fileName)
	if err != nil {
		return FansubInfo{}, err
	}
	fansubName = tokens[0]
	tokens = tokens[1:] // Ignore fansub name

	addedEncodings := map[string]struct{}{}

	tryWriteEncoding := func(enc string) {
		// Don't add duplicate encodings
		if _, exists := addedEncodings[enc]; !exists {
			encoding.WriteString(enc)
			addedEncodings[enc] = struct{}{}
		}
	}

	for i, t := range tokens {
		if t == "" {
			continue
		}

		token := t
		normalizedToken := normalizeToken(t)

		enc, multiEnc := getFansubEncoding(normalizedToken, i, tokens)
		tryWriteEncoding(enc)

		if len(multiEnc) > 0 {
			for i, enc = range multiEnc {
				enc, _ = getFansubEncoding(enc, i, multiEnc)
				tryWriteEncoding(enc)
			}
		}

		hasMetaData := func() bool {
			return encoding.Len() > 0 || source.Len() > 0 || episode != "" || season != ""
		}

		source.WriteString(getFansubSource(normalizedToken))

		// Assume episode format is always at beginning of file name
		if season == "" && episode == "" && !hasMetaData() {
			season, episode = getFansubEpisode(token, i, tokens)
		}

		// Assume "batch" always comes after some meta data
		if hasMetaData() && normalizedToken == "batch" {
			return FansubInfo{}, fmt.Errorf("batch files not supported")
		}

		// Assume title is always before meta-data
		if !hasMetaData() {
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

func getFansubEncoding(s string, index int, tokens []string) (string, []string) {
	// Catch "<enc>.<enc>" where multiple encodings can be separated by ellipses
	var possibleEncodings []string
	if strings.Contains(s, ".") {
		possibleEncodings = strings.Split(s, ".")
	}

	if val, exists := fansubEncodingMap[s]; exists {
		// Catch "AAC 2.0" formatting
		if s == "aac" && index+1 < len(tokens) {
			if utils.RemoveBrackets(tokens[index+1]) == "2.0" {
				return val + "2.0 ", possibleEncodings
			}
		}

		// Catch "TrueHD 7.1" formatting
		if s == "truehd" {
			nextToken := normalizeToken(tokens[index+1])
			if nextToken == "7" {
				return val + "7.1 ", possibleEncodings
			}
		}
		return val + " ", possibleEncodings
	}

	// Catch "H 264" formatting
	if s == "264" {
		lastToken := normalizeToken(tokens[index-1])
		if lastToken == "h" {
			return "H.264 ", possibleEncodings
		}
	}

	// Catch "H 265" formatting
	if s == "265" {
		lastToken := normalizeToken(tokens[index-1])
		if lastToken == "h" {
			return fansubEncodingMap["x265"] + " ", possibleEncodings
		}
	}

	// Catch "AAC2 0" formatting
	if s == "aac2" {
		nextToken := normalizeToken(tokens[index+1])
		if nextToken == "0" {
			return fansubEncodingMap["aac2.0"] + " ", possibleEncodings
		}

	}

	// Catch "DDP5 1" formatting
	if s == "ddp5" {
		nextToken := normalizeToken(tokens[index+1])
		if nextToken == "1" {
			return fansubEncodingMap["ddp5.1"] + " ", possibleEncodings
		}
	}

	// Catch "DDP2 0" formatting
	if s == "ddp2" {
		nextToken := normalizeToken(tokens[index+1])
		if nextToken == "0" {
			return fansubEncodingMap["ddp2.0"] + " ", possibleEncodings
		}
	}

	resolutions := []string{
		"240p", "360p", "480p", "540p", "720p", "1080p", "1440p", "2160p",
	}

	for _, res := range resolutions {
		if res == s {
			return res + " ", possibleEncodings
		}
	}

	return "", possibleEncodings
}

func getFansubSource(s string) string {
	if source, exists := fansubSourceMap[s]; exists {
		return source + " "
	}
	return ""
}

func getFansubEpisode(s string, index int, tokens []string) (season string, episode string) {
	if strings.TrimSpace(s) == "" {
		return
	}

	trimEpVersion := func(episode string) string {
		return strings.TrimSuffix(episode, "v2")
	}

	// Catch "Season # - #" for season & episode
	if s == "Season" && index+3 < len(tokens) && tokens[index+2][0] == '-' {
		if utils.IsNumber(tokens[index+1]) && utils.IsNumber(tokens[index+3]) {
			return tokens[index+1], tokens[index+3]
		}
	}

	// Catch "S##E##" for season & episode
	if s[0] == 'S' && (len(s) == 6 || len(s) == 8) {
		if utils.IsNumber(s[1:3]) && utils.IsNumber(trimEpVersion(s[4:])) {
			return s[1:3], s[4:]
		}
	}

	// Catch "S# - #" for season & episode
	if s[0] == 'S' && (len(s) == 2 || len(s) == 3) && index+2 < len(tokens) {
		if utils.IsNumber(s[1:]) && utils.IsNumber(trimEpVersion(tokens[index+2])) {
			return s[1:], tokens[index+2]
		}
	}

	// Catch "- #" for episode
	if s[0] == '-' && index+1 < len(tokens) {
		if utils.IsNumber(trimEpVersion(tokens[index+1])) {
			return "", tokens[index+1]
		}
	}

	// Catch "EP#" for episode
	if len(s) > 2 && s[:2] == "EP" {
		if utils.IsNumber(trimEpVersion(s[2:])) {
			return "", s[2:]
		}
	}

	// Catch "#<suffix> Season - #" for season & episode
	if len(s) > 2 && len(s) < 5 && index+3 < len(tokens) {
		if strings.ToLower(tokens[index+1]) == "season" {
			season = strings.TrimSuffix(s, "st")
			season = strings.TrimSuffix(season, "nd")
			season = strings.TrimSuffix(season, "rd")
			season = strings.TrimSuffix(season, "th")

			if utils.IsNumber(season) && utils.IsNumber(trimEpVersion(tokens[index+3])) {
				return season, tokens[index+3]
			}
		}
	}

	// Catch "<title> # [<meta-data>" for episode
	if utils.IsNumber(trimEpVersion(s)) && index+1 < len(tokens) {
		if tokens[index+1][0] == '[' {
			return "", s
		}
	}

	// Catch "S#" for episode
	if s[0] == 'S' {
		if utils.IsNumber(s[1:]) {
			return s[1:], ""
		}
	}

	return
}

func getFansubTokens(fileName string) ([]string, error) {
	spaceCount := strings.Count(fileName, " ")
	ellipsesCount := strings.Count(fileName, ".")
	underscoreCount := strings.Count(fileName, "_")

	var fansubName string
	if fileName[0] == '[' {
		fansubEndIndex := strings.Index(fileName, "]")
		fansubName = fileName[1:strings.Index(fileName, "]")]
		fileName = strings.TrimSpace(fileName[fansubEndIndex+1:])
	}

	// Try to parse inconsistent ellipses separators
	if ellipsesCount >= spaceCount && spaceCount > 0 {
		var lastToken string
		tokens := strings.Split(fileName, " ")
		if strings.Contains(tokens[0], ".") {
			tokens = strings.Split(tokens[0], ".")
			lastToken = tokens[len(tokens)-1]
			if strings.Contains(lastToken, "-") {
				fansubName = strings.Split(lastToken, "-")[1]
			}
		} else {
			return []string{}, fmt.Errorf("unsupported separator format")
		}
		tokens = tokens[:len(tokens)-1]
		tokens = append(tokens, strings.ReplaceAll(lastToken, "-"+fansubName, ""))
		tokens = append([]string{fansubName}, tokens...)
		return tokens, nil
	}

	if spaceCount > ellipsesCount && spaceCount > underscoreCount {
		fileName = utils.ReplaceAll(fileName, newBracketReplaceMap(" "))
		tokens := strings.Split(fileName, " ")
		tokens = append([]string{fansubName}, tokens...)
		return tokens, nil
	}

	if ellipsesCount > underscoreCount {
		tokens := strings.Split(fileName, ".")
		lastToken := tokens[len(tokens)-1]
		// Look for fansub group at end of file name
		if strings.Contains(lastToken, "-") {
			tokens = tokens[:len(tokens)-1]
			lastTokens := strings.Split(lastToken, "-")
			if len(lastTokens) != 2 {
				return []string{}, fmt.Errorf("unsupported end-token format")
			}
			tokens = append([]string{fmt.Sprintf("%s", lastTokens[1])}, tokens...)
			tokens = append(tokens, lastTokens[0])
		}
		return tokens, nil
	}

	fileName = utils.ReplaceAll(fileName, newBracketReplaceMap("_"))
	tokens := strings.Split(fileName, "_")
	tokens = append([]string{fansubName}, tokens...)
	return tokens, nil
}

func newBracketReplaceMap(replacement string) map[string]string {
	return map[string]string{
		"][": replacement,
		")(": replacement,
		"](": replacement,
		")[": replacement,
	}
}

func normalizeToken(token string) string {
	return utils.RemoveBrackets(strings.ToLower(token))
}
