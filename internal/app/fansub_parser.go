package app

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Jaeiya/koshime/internal/utils"
)

var fansubEncodingMap = map[string]string{
	"1920x1080":  "1080p",
	"1920x1080i": "1080i",
	"1440x1080":  "1080p-Squeezed",
	"1280x720":   "720p",
	"854x480":    "480p",
	"640x360":    "360p",

	"av1":        "AV1",
	"aac":        "AAC",
	"aac2.0":     "AAC 2.0",
	"hevc-10bit": "HEVC 10-bit",
	"hevc":       "HEVC",
	"h.265":      "HEVC",
	"x265":       "HEVC",
	"avc":        "H.264",
	"h.264":      "H.264",
	"x264":       "H.264",

	"hdr": "HDR",

	"sdr":    "SDR",
	"ddp":    "DolbyDigital+",
	"ddp2.0": "DolbyDigital+2.0",
	"ddp5.1": "DolbyDigital+5.1",
	"dv":     "DolbyVision",
	"atmos":  "DolbyAtmos",

	"truehd":    "TrueHD",
	"truehd7.1": "TrueHD 7.1",
	"opus":      "Opus",
	"mp3":       "Mp3",
	"flac":      "FLAC",
	"e-ac-3":    "EAC3",
	"e-ac3":     "EAC3",
	"eac3":      "EAC3",
	"eac-3":     "EAC3",
	"7.1":       "7.1",
	"2.0":       "2.0",

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

var versionMap = map[string]struct{}{
	// Realistically there should never be more than 4 versions, but just in case...
	"v0": {}, "v1": {}, "v2": {}, "v3": {}, "v4": {}, "v5": {}, "v6": {}, "v7": {},
}

var ErrBatchFile = fmt.Errorf("batch files not supported")

var fansubExtMap = map[string]struct{}{
	".mp4": {},
	".mkv": {},
	".avi": {},
	".mpg": {},
	".wmv": {},
	".ts":  {},
}

type FansubFileInfo struct {
	Fansub   string
	Title    string
	Encoding string
	Season   string
	Episode  string
	Version  int
	Source   string
	Filename string
}

type FansubParser struct{}

// IsSupported returns whether or not the provided file name
// has a supported extension. Files without extensions
// are treated as having an unsupported extension.
func (FansubParser) IsSupported(fileName string) bool {
	_, exists := fansubExtMap[filepath.Ext(fileName)]
	return exists
}

func (fp FansubParser) Parse(fileName string) (FansubFileInfo, error) {
	ext := filepath.Ext(fileName)
	if _, hasExt := fansubExtMap[ext]; hasExt {
		fileName = strings.TrimSuffix(fileName, ext)
	}

	if !strings.ContainsAny(fileName, " ._") {
		return FansubFileInfo{}, fmt.Errorf("unsupported file name")
	}

	var fansubName, episode, season string
	var version int
	var encoding, source, title strings.Builder

	tokens, err := fp.getTokens(fileName)
	if err != nil {
		return FansubFileInfo{}, err
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
		normalizedToken := fp.normalizeToken(t)

		enc, multiEnc := fp.getEncoding(normalizedToken, i, tokens)
		tryWriteEncoding(enc)

		if len(multiEnc) > 0 {
			for i, enc = range multiEnc {
				enc, _ = fp.getEncoding(enc, i, multiEnc)
				tryWriteEncoding(enc)
			}
		}

		hasMetaData := func() bool {
			return encoding.Len() > 0 || source.Len() > 0 || episode != "" || season != ""
		}

		source.WriteString(fp.getSource(normalizedToken))

		// Assume episode format is always at beginning of file name
		if season == "" && episode == "" && !hasMetaData() {
			season, episode = fp.getEpisode(token, i, tokens)
		}

		if version == 0 && (len(season) > 0 || len(episode) > 0) {
			version, err = fp.getVersion(token)
			if err != nil {
				return FansubFileInfo{}, err
			}
		}

		// Assume "batch" always comes after some meta data
		if hasMetaData() && normalizedToken == "batch" {
			return FansubFileInfo{}, ErrBatchFile
		}

		// Assume title is always before meta-data
		if !hasMetaData() {
			title.WriteString(t)
			title.WriteString(" ")
		}

	}

	info := FansubFileInfo{}

	info.Fansub = fansubName
	info.Episode = episode
	info.Version = version
	info.Season = season
	info.Title = strings.TrimRight(title.String(), "- ~")
	info.Encoding = strings.TrimSpace(encoding.String())
	info.Source = strings.TrimSpace(source.String())

	info.Filename = fileName
	if _, hasExt := fansubExtMap[ext]; hasExt {
		info.Filename = fileName + ext
	}

	return info, nil
}

func (fp FansubParser) getEncoding(s string, index int, tokens []string) (string, []string) {
	// Catch "<enc>.<enc>" where multiple encodings can be separated by ellipses
	var possibleEncodings []string
	if strings.Contains(s, ".") {
		possibleEncodings = strings.Split(s, ".")
	}

	hasNextToken := index+1 < len(tokens)

	if val, exists := fansubEncodingMap[s]; exists {
		// Catch "AAC.2.0" formatting
		if s == "aac" && hasNextToken {
			nextToken := fp.normalizeToken(tokens[index+1])
			if nextToken == "2" {
				return val + " 2.0 ", possibleEncodings
			}
		}

		// Catch "TrueHD.7.1" formatting
		if s == "truehd" && hasNextToken {
			nextToken := fp.normalizeToken(tokens[index+1])
			if nextToken == "7" {
				return val + " 7.1 ", possibleEncodings
			}
		}
		return val + " ", possibleEncodings
	}

	// Catch "H 264" formatting
	if s == "264" || s == "264-varyg" {
		lastToken := fp.normalizeToken(tokens[index-1])
		if lastToken == "h" {
			return "H.264 ", possibleEncodings
		}
	}

	// Catch "H 265" formatting
	if s == "265" {
		lastToken := fp.normalizeToken(tokens[index-1])
		if lastToken == "h" {
			return fansubEncodingMap["x265"] + " ", possibleEncodings
		}
	}

	// Catch "AAC2 0" formatting
	if s == "aac2" && hasNextToken {
		nextToken := fp.normalizeToken(tokens[index+1])
		if nextToken == "0" {
			return fansubEncodingMap["aac2.0"] + " ", possibleEncodings
		}

	}

	// Catch "DDP5 1" formatting
	if s == "ddp5" && hasNextToken {
		nextToken := fp.normalizeToken(tokens[index+1])
		if nextToken == "1" {
			return fansubEncodingMap["ddp5.1"] + " ", possibleEncodings
		}
	}

	// Catch "DDP2 0" formatting
	if s == "ddp2" && hasNextToken {
		nextToken := fp.normalizeToken(tokens[index+1])
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

func (FansubParser) getSource(s string) string {
	if source, exists := fansubSourceMap[s]; exists {
		return source + " "
	}
	return ""
}

func (FansubParser) getEpisode(
	s string,
	index int,
	tokens []string,
) (string, string) {
	if strings.TrimSpace(s) == "" {
		return "", ""
	}

	trimEpVersion := func(episode string) string {
		if len(episode) < 3 {
			return episode
		}

		v := episode[len(episode)-2:]
		if _, exists := versionMap[v]; exists {
			return strings.TrimSuffix(episode, v)
		}

		return episode
	}

	// Catch "Season # - #" for season & episode
	if s == "Season" && index+3 < len(tokens) && tokens[index+2][0] == '-' {
		episode := trimEpVersion(tokens[index+3])
		if utils.IsNumber(tokens[index+1]) && utils.IsNumber(episode) {
			return tokens[index+1], episode
		}
	}

	// Catch "S##E##" for season & episode
	if s[0] == 'S' && (len(s) == 6 || len(s) == 8) {
		episode := trimEpVersion(s[4:])
		if utils.IsNumber(s[1:3]) && utils.IsNumber(episode) {
			return s[1:3], episode
		}
	}

	// Catch "S# - #" for season & episode
	if s[0] == 'S' && (len(s) == 2 || len(s) == 3) && index+2 < len(tokens) {
		episode := trimEpVersion(tokens[index+2])
		if utils.IsNumber(s[1:]) && utils.IsNumber(episode) {
			return s[1:], episode
		}
	}

	// Catch "- #" for episode
	if s[0] == '-' && index+1 < len(tokens) {
		episode := trimEpVersion(tokens[index+1])
		if utils.IsNumber(episode) {
			return "", episode
		}
	}

	// Catch "EP#" for episode
	if len(s) > 2 && s[:2] == "EP" {
		episode := trimEpVersion(s[2:])
		if utils.IsNumber(episode) {
			return "", episode
		}
	}

	// Catch "#<suffix> Season - #" for season & episode
	if len(s) > 2 && len(s) < 5 && index+3 < len(tokens) {
		if strings.ToLower(tokens[index+1]) == "season" {
			season := strings.TrimSuffix(s, "st")
			season = strings.TrimSuffix(season, "nd")
			season = strings.TrimSuffix(season, "rd")
			season = strings.TrimSuffix(season, "th")
			episode := trimEpVersion(tokens[index+3])

			if utils.IsNumber(season) && utils.IsNumber(episode) {
				return season, episode
			}
		}
	}

	// Catch "<title> # [<meta-data>" for episode
	if utils.IsNumber(trimEpVersion(s)) && index+1 < len(tokens) {
		if tokens[index+1][0] == '[' {
			return "", trimEpVersion(s)
		}
	}

	// Catch "S#" for season
	if s[0] == 'S' {
		if utils.IsNumber(s[1:]) {
			return s[1:], ""
		}
	}

	return "", ""
}

func (FansubParser) getVersion(s string) (int, error) {
	if len(s) < 2 {
		return 0, nil
	}

	version := s[len(s)-2:]
	if _, exists := versionMap[s[len(s)-2:]]; exists {
		v, err := strconv.Atoi(version[1:])
		if err != nil {
			return 0, fmt.Errorf("could not parse episode version: %w", err)
		}
		return v, nil
	}

	return 0, nil
}

func (fp FansubParser) getTokens(fileName string) ([]string, error) {
	spaceCount := strings.Count(fileName, " ")
	ellipsesCount := strings.Count(fileName, ".")
	underscoreCount := strings.Count(fileName, "_")

	var fansubName string
	if fileName[0] == '[' {
		fansubEndIndex := strings.Index(fileName, "]")
		fansubName = fileName[1:strings.Index(fileName, "]")]
		fileName = strings.TrimSpace(fileName[fansubEndIndex+1:])
	}

	// Explicit fansub group detection
	if fansubName == "" {
		if strings.Contains(fileName, "264-VARYG") {
			fansubName = "VARYG"
		}
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
		fileName = utils.ReplaceAll(fileName, fp.newBracketReplaceMap(" "))
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
			tokens = append([]string{lastTokens[1]}, tokens...)
			tokens = append(tokens, lastTokens[0])
		}
		return tokens, nil
	}

	fileName = utils.ReplaceAll(fileName, fp.newBracketReplaceMap("_"))
	tokens := strings.Split(fileName, "_")
	tokens = append([]string{fansubName}, tokens...)
	return tokens, nil
}

func (FansubParser) newBracketReplaceMap(replacement string) map[string]string {
	return map[string]string{
		"][": replacement,
		")(": replacement,
		"](": replacement,
		")[": replacement,
	}
}

func (FansubParser) normalizeToken(token string) string {
	return utils.RemoveBrackets(strings.ToLower(token))
}
