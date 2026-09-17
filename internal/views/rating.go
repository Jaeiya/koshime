package views

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Jaeiya/koshime/internal/kitsu"
	"github.com/Jaeiya/koshime/internal/ui"
	"github.com/Jaeiya/koshime/internal/utils"
)

type (
	RatingDeclinedMsg  struct{} // User declined to rate the anime
	RatingCompletedMsg struct{} // Rating an anime has succeeded
	finishedRatingMsg  struct{} // Rating command has finished executing
)

type RatingModel struct {
	windowSize tea.WindowSizeMsg
	ui         struct {
		consent ui.ConsentModel
		loader  ui.LoaderModel
		input   textinput.Model
	}
	animeTitle  string
	header      string
	libID       string // Anime library ID
	authToken   string
	inputStatus string
}

func newRatingModel(libID, authToken string) RatingModel {
	m := RatingModel{}
	m.ui.consent.Activate()
	m.ui.loader = ui.NewLoader()
	m.ui.input = ui.NewTextInput()
	m.ui.input.SetWidth(7)
	m.ui.input.Placeholder = "<rating>"
	m.libID = libID
	m.authToken = authToken
	return m
}

func (m RatingModel) Init() tea.Cmd {
	return nil
}

func (m RatingModel) Update(msg tea.Msg) (RatingModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg

	case tea.KeyPressMsg:
		if m.ui.loader.IsLoading() {
			return m, nil
		}
		m, cmd = m.onKeyPress(msg)
		cmds = append(cmds, cmd)

	case finishedRatingMsg:
		m.ui.loader.Stop()
		return m, func() tea.Msg { return RatingCompletedMsg{} }

	case error:
		if m.ui.loader.IsLoading() {
			m.ui.loader.Stop()
		}
	}

	if m.ui.loader.IsLoading() {
		m.ui.loader, cmd = m.ui.loader.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.ui.consent.IsActive() {
		m.ui.consent = m.ui.consent.Update(msg)
	} else {
		m.ui.input, cmd = m.ui.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m RatingModel) onKeyPress(msg tea.KeyPressMsg) (RatingModel, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, ui.KeyMap.EscBack):
		return m, func() tea.Msg { return RatingDeclinedMsg{} }

	case key.Matches(msg, ui.KeyMap.Select):
		if m.ui.consent.IsActive() {
			if m.ui.consent.Select() == ui.No {
				return m, func() tea.Msg { return RatingDeclinedMsg{} }
			}
			return m, nil
		}

		input := strings.TrimSpace(m.ui.input.Value())
		rating, err := m.parseRating(input)
		if err != nil {
			m.inputStatus = err.Error()
			return m, nil
		}
		m.ui.loader, cmd = m.ui.loader.Start("Rating Anime")
		return m, tea.Batch(cmd, m.rateAnime(rating))

	default: // Clear status when user types
		if m.inputStatus != "" {
			m.inputStatus = ""
		}
	}

	return m, cmd
}

func (m RatingModel) View() tea.View {
	v := tea.NewView("")

	if m.ui.loader.IsLoading() {
		v.Content = ui.Style.MarginTop(1).Render(m.ui.loader.View())
		return v
	}

	if m.ui.consent.IsActive() {
		v.Content = lipgloss.JoinVertical(
			lipgloss.Left,
			ui.DisplaySubTitle(m.header, "Rating Consent"),
			ui.DisplayText([]string{
				"You're about to set the rating for the following anime:",
				fmt.Sprintf(";dc;%s;x;", m.animeTitle),
			}, 1, 1, 1),
			m.ui.consent.View(
				ui.ConsentStyle.Render("Would you like to continue?"),
			),
		)
		return v
	}

	v.Cursor = m.ui.input.Cursor()

	errStr := m.inputStatus
	if errStr != "" {
		errStr = ui.Style.MarginLeft(3).
			Foreground(lipgloss.BrightRed).
			Render(fmt.Sprintf("!! %s", errStr))
	}

	v.Content = lipgloss.JoinVertical(
		lipgloss.Left,
		ui.DisplaySubTitle(m.header, "Rate Anime"),
		ui.DisplayText([]string{
			`A rating must be between ;b;1;x; and ;b;10;x;. Where ;b;10;x;
is the highest and ;b;1;x; is the lowest.`,
			`Valid values can consist of ;b;0.5;x; increments. For instance
;b;1.5;x; or ;b;6.5;x; but ;y;not;x; ;b;7.3;x; or ;b;3.9;x;.`,
			`Enter your rating below:`,
		}, 1, 1, 1),
		m.ui.input.View(),
		"",
		errStr,
	)

	v.Cursor.Y = lipgloss.Height(v.Content) - 3
	v.Cursor.Shape = tea.CursorBar

	return v
}

func (m RatingModel) ShortHelp() []key.Binding {
	return []key.Binding{ui.KeyMap.Up, ui.KeyMap.Down, ui.KeyMap.EscBack}
}

func (m RatingModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func (m *RatingModel) SetAnimeTitle(title string) {
	m.animeTitle = title
}

func (m *RatingModel) SetHeader(header string) {
	m.header = header
}

func (m RatingModel) parseRating(input string) (int, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return -1, fmt.Errorf("missing value")
	}

	if !strings.Contains(input, ".") {
		rating, err := strconv.Atoi(input)
		if err != nil {
			return -1, fmt.Errorf("invalid rating")
		}
		if rating > 10 || rating < 1 {
			return -1, fmt.Errorf("rating exceeds expected range")
		}
		return utils.ToTwentyRating(float64(rating)), nil
	}

	floatParts := strings.Split(input, ".")
	isInvalidFloat := len(floatParts) > 2 || !utils.IsNumber(floatParts[0]) ||
		!utils.IsNumber(floatParts[1])
	if isInvalidFloat {
		return -1, fmt.Errorf("invalid float value")
	}

	leftInt, _ := strconv.Atoi(floatParts[0])
	rightInt, _ := strconv.Atoi(floatParts[1])

	if leftInt > 10 || leftInt < 1 || (leftInt == 10 && rightInt > 0) {
		return -1, fmt.Errorf("rating exceeds expected range")
	}

	if rightInt > 0 && rightInt != 5 {
		return -1, fmt.Errorf("invalid fraction; increment by 0.5")
	}

	rating := float64(leftInt)
	if rightInt > 0 {
		rating += 0.5
	}
	return utils.ToTwentyRating(rating), nil
}

func (m RatingModel) rateAnime(rating int) tea.Cmd {
	return func() tea.Msg {
		if _, err := kitsu.RateAnime(m.libID, m.authToken, rating); err != nil {
			return fmt.Errorf("failed to rate anime: %w", err)
		}
		return finishedRatingMsg{}
	}
}
