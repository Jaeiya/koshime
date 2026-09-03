package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/Jaeiya/koshime/internal/logger"
)

type LoaderModel struct {
	spinner spinner.Model
	text    string
	active  bool
}

func NewLoader() LoaderModel {
	s := spinner.New(spinner.WithSpinner(spinner.Spinner{
		Frames: []string{"⠋", "⠙", "⠚", "⠞", "⠖", "⠦", "⠴", "⠲", "⠳", "⠓"},
		FPS:    time.Second / 10,
	}))

	return LoaderModel{spinner: s}
}

func (m LoaderModel) Init() tea.Cmd {
	logger.Log(logger.Debug, "Init(): send spinner tick")
	if m.IsLoading() {
		return m.spinner.Tick
	}
	return nil
}

func (m LoaderModel) Update(msg tea.Msg) (LoaderModel, tea.Cmd) {
	var cmd tea.Cmd
	if m.active {
		m.spinner, cmd = m.spinner.Update(msg)
	}
	return m, cmd
}

func (m LoaderModel) View() string {
	spinnerStr := SpinnerStyle.Render(strings.Repeat(m.spinner.View(), 3))
	return TextStyle.Render(
		fmt.Sprintf(
			"%s %s %s",
			spinnerStr,
			LoadingStyle.Render(m.text),
			spinnerStr,
		),
	)
}

func (m LoaderModel) IsLoading() bool {
	return m.active
}

func (m LoaderModel) Start(text string) (LoaderModel, tea.Cmd) {
	logger.Log(logger.Debug, "starting loader: %s", text)
	m.active = true
	if text != "" {
		m.text = text
	}
	return m, m.spinner.Tick
}

func (m *LoaderModel) Stop() {
	logger.Log(logger.Debug, "stopping loader: %s", m.text)
	m.active = false
}
