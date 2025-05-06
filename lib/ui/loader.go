package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/v2/spinner"
	tea "github.com/charmbracelet/bubbletea/v2"
)

type loaderModel struct {
	spinner spinner.Model
	text    string
	active  bool
}

func NewLoader() loaderModel {
	s := spinner.New(spinner.WithSpinner(spinner.Spinner{
		Frames: []string{"⠋", "⠙", "⠚", "⠞", "⠖", "⠦", "⠴", "⠲", "⠳", "⠓"},
		FPS:    time.Second / 10,
	}))

	return loaderModel{spinner: s}
}

func (m loaderModel) Init() tea.Cmd {
	return nil
}

func (m loaderModel) Update(msg tea.Msg) (loaderModel, tea.Cmd) {
	var cmd tea.Cmd
	if m.active {
		m.spinner, cmd = m.spinner.Update(msg)
	}
	return m, cmd
}

func (m loaderModel) View() string {
	spinnerStr := spinnerStyle.Render(strings.Repeat(m.spinner.View(), 3))
	return textStyle.Render(
		fmt.Sprintf(
			"%s %s %s",
			spinnerStr,
			loadingStyle.Render(m.text),
			spinnerStr,
		),
	)
}

func (m loaderModel) IsLoading() bool {
	return m.active
}

func (m *loaderModel) SetLoadingState(b bool) {
	m.active = true
}

func (m *loaderModel) SetText(s string) {
	m.text = s
}

func (m loaderModel) Start() tea.Msg {
	return m.spinner.Tick()
}

func (m *loaderModel) Stop() {
	m.active = false
}
