package main

import (
	"fmt"
	"path/filepath"

	"github.com/Jaeiya/koshime/lib/views"
	tea "github.com/charmbracelet/bubbletea/v2"
)

var (
	errUserAborted = fmt.Errorf("user aborted initialization")
	dbFilePath     = filepath.Join("./", "koshime.db")
)

func main() {
	m, err := views.New(dbFilePath)
	if err != nil {
		panic(err)
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		panic(err)
	}

	// 	fmt.Println("    Date:", entry.Date.Format("01/02/2006 3:04 PM"))
}
