package main

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	lib "github.com/Jaeiya/koshime/internal"
	"github.com/Jaeiya/koshime/internal/views"
)

var (
	version   string
	commitSha string
	buildDate string
)

func main() {
	lib.Version = version
	lib.CommitHash = commitSha
	lib.BuildDate = buildDate

	m, err := views.New()
	if err != nil {
		panic(err)
	}

	// Cleanup residual cursor issues
	defer fmt.Print("\x1b[0 q")

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		panic(err)
	}
}
