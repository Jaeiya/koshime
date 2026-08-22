package views

import (
	"fmt"
	"net/url"
	"os"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/Jaeiya/koshime/internal/kitsu"
	"github.com/Jaeiya/koshime/internal/ui"
	"github.com/Jaeiya/koshime/internal/utils"
	"github.com/charmbracelet/x/term"
)

type DisplayMode uint8

const (
	Simple = DisplayMode(iota)
	Extended
	All
)

var MaxSynopsisLen = func() int {
	_, h, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		return 500
	}

	const menuSize = 23.0 // Estimation of menu size
	const padding = 0.20  // Prevent filling entire screen
	const charsPerLine = 40

	ratio := (1.0 - (menuSize / float64(h))) - padding
	return int((float64(h) * ratio)) * charsPerLine
}()

type AnimeDisplayModel struct {
	keys struct {
		simple   key.Binding
		extended key.Binding
		all      key.Binding
	}
	displayMode DisplayMode
}

func NewAnimeDisplayModel() *AnimeDisplayModel {
	m := &AnimeDisplayModel{}
	m.keys.simple = key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "view simple"))
	m.keys.extended = key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "view extended"))
	m.keys.all = key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "view all"))
	return m
}

func (m *AnimeDisplayModel) Update(msg tea.Msg) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.simple):
			switch m.displayMode {
			case Simple:
				m.displayMode = Extended
			case Extended:
				m.displayMode = All
			case All:
				m.displayMode = Simple
			}
		}
	}
}

func (m AnimeDisplayModel) View(ai kitsu.Anime) string {
	return m.DisplayAnimeInfo(ai, m.displayMode)
}

func (m AnimeDisplayModel) ShortHelp() []key.Binding {
	switch m.displayMode {
	case Simple:
		return []key.Binding{m.keys.extended}
	case Extended:
		return []key.Binding{m.keys.all}
	case All:
		return []key.Binding{m.keys.simple}
	default:
		return []key.Binding{}
	}
}

func (m AnimeDisplayModel) DisplayAnimeInfo(info kitsu.Anime, mode DisplayMode) string {
	headers := []string{
		utils.ColorText(";g;Title"),
		utils.ColorText(";dc;Cannon"),
	}

	items := []string{
		info.ENG_Title,
		info.JPN_Title,
	}

	if items[0] == "" {
		items[0] = info.JPN_Title
	}

	// Accept only utf8 titles
	altTitles := make([]string, 0, len(info.AltTitles))
	for i := range len(info.AltTitles) {
		if utils.HasNonASCII(info.AltTitles[i]) {
			continue
		}
		altTitles = append(altTitles, info.AltTitles[i])
	}

	if len(altTitles) > 0 {
		for range len(altTitles) {
			headers = append(headers, utils.ColorText(";db;AltTitle"))
			if Simple == mode {
				break
			}
		}
	} else {
		headers = append(headers, utils.ColorText(";db;AltTitle"))
	}

	const charLimit = 60

	if Simple == mode {
		for i, item := range items {
			if len(item) > charLimit {
				items[i] = item[:charLimit-2] + ".."
			}
		}
		if len(altTitles) > 0 {
			title := altTitles[0]
			if len(title) > charLimit {
				title = title[:charLimit-2] + ".."
			}
			items = append(items, title)
		} else {
			items = append(items, utils.ColorText(";bk;None"))
		}
	}

	if Extended == mode || All == mode {
		if len(altTitles) > 0 {
			for _, title := range altTitles {
				if utils.HasNonASCII(title) {
					continue
				}
				items = append(items, title)
			}
		} else {
			items = append(items, utils.ColorText(";bk;None"))
		}
	}

	headers = append(headers, utils.ColorText(";dc;Status"))
	items = append(items, utils.ColorText(";b;"+info.Status))

	headers = append(headers, utils.ColorText(";y;Type"))
	items = append(items, utils.ColorText(";c;"+info.Type))

	totalEpsStr := utils.ColorText(";bk;Unknown")
	if info.Episodes > 0 {
		totalEpsStr = fmt.Sprintf(";m;%d", info.Episodes)
	}

	if info.Progress > -1 {
		headers = append(headers, utils.ColorText(";y;Progress"))
		items = append(items, utils.ColorText(
			fmt.Sprintf(";dg;%d ;y;/ %s", info.Progress, totalEpsStr),
		))
	} else {
		headers = append(headers, utils.ColorText(";dc;Episodes"))
		items = append(items, utils.ColorText(fmt.Sprintf(";m;%s", totalEpsStr)))
	}

	avgRating := utils.ColorText(fmt.Sprintf(";w;%s", info.AvgRating))
	if info.AvgRating == "" {
		avgRating = utils.ColorText(";bk;Not Calculated")
	}
	headers = append(headers, utils.ColorText(";dc;AvgRating"))
	items = append(items, avgRating)

	hasFeed := info.QbtFeed.Name != ""

	headers = append(headers, utils.ColorText(";dc;RSSFeed"))
	switch mode {
	case Simple:
		if hasFeed {
			items = append(items, utils.ColorText(";g;Yes"))
		} else {
			items = append(items, utils.ColorText(";m;No"))
		}
	case Extended, All:
		if hasFeed {
			uri := info.QbtFeed.RuleURI
			if mode == Extended && len(uri) > charLimit {
				uri = uri[:charLimit-2] + ".."
			}
			items = append(items, utils.ColorText(fmt.Sprintf(";g;%s", uri)))
		} else {
			items = append(items, utils.ColorText(";m;No"))
		}
	}

	link, _ := url.JoinPath(kitsu.KitsuDomain, "anime", info.Slug)

	if All == mode {
		headers = append(
			headers,
			utils.ColorText(";x;Link"),
			utils.ColorText(";dc;Synopsis"),
		)
		if len(info.Synopsis) > MaxSynopsisLen {
			info.Synopsis = info.Synopsis[:MaxSynopsisLen-3] + "..."
		}
		items = append(
			items,
			utils.ColorText(";dy;"+link),
			info.Synopsis,
		)
	}

	return ui.DisplayPropValue(headers, items)
}
