package main

import (
	"fmt"

	"github.com/Jaeiya/koshime/lib/views"
	tea "github.com/charmbracelet/bubbletea/v2"
)

func main() {
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
