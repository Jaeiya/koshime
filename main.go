package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Jaeiya/koshime/internal/app"
	"github.com/Jaeiya/koshime/internal/logger"
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

	if err := logger.SetLogLevel(logger.Debug); err != nil {
		printFatal(err.Error(), "Error setting log level; report immediately!")
	}
	defer logger.CloseLog()

	logger.Log(logger.Info, "initialize main view")
	m, err := views.New()
	if err != nil {
		printFatal(
			err.Error(),
			"Error initializing the main view; report immediately!",
		)
	}

	logger.Log(logger.Info, "starting program")
	p := tea.NewProgram(m)
	updatedModel, err := p.Run()
	if err != nil {
		printFatal(
			err.Error(),
			"Unexpected program error; report immediately!",
		)
	}

	model, _ := updatedModel.(views.Model)
	if model.FatalErr.Msg != "" {
		model.FatalErr.Desc += "; report immediately!"
		printFatal(model.FatalErr.Msg, model.FatalErr.Desc)
	}

	if model.HasAborted {
		logger.Log(logger.Info, "user aborted application")
		fmt.Printf(
			"%s",
			ui.DisplayText(
				[]string{utils.ColorText(";g;>>> ;y;User Aborted Setup ;g;<<<")},
				0, 1, 1,
			),
		)
	}
}

func printFatal(msg, desc string) {
	logger.Log(logger.Error, msg)
	errStyle := lipgloss.NewStyle().Width(55).MarginLeft(2)
	msg = errStyle.Render(msg)
	desc = errStyle.Render(desc)
	fmt.Printf(utils.ColorText(
		"\n  ;g;>>> ;y;Fatal Error ;g;<<<;x;\n\n;dr;%s;x;\n\n;dy;%s;x;\n\n",
	), msg, desc)
	os.Exit(1)
}
