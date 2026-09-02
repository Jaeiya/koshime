package main

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/Jaeiya/koshime/internal/app"
	"github.com/Jaeiya/koshime/internal/ui"
	"github.com/Jaeiya/koshime/internal/utils"
	"github.com/Jaeiya/koshime/internal/views"
)

var (
	version   string
	commitSha string
	buildDate string
)

func main() {
	app.Version = version
	app.CommitHash = commitSha
	app.BuildDate = buildDate

	m, err := views.New()
	if err != nil {
		panic(err)
	}

	p := tea.NewProgram(m)
	updatedModel, err := p.Run()
	if err != nil {
		panic(err)
	}

	model, _ := updatedModel.(views.Model)
	if model.HasAborted {
		fmt.Printf(
			"%s",
			ui.DisplayText(
				[]string{utils.ColorText(";g;>>> ;y;User Aborted Setup ;g;<<<")},
				0, 1, 1,
			),
		)
	}
}
