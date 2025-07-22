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

	// db, err := database.NewDatabase(nil)
	// if err != nil {
	// 	panic(err)
	// }
	// db.Load()

	// var ff lib.FansubFilter
	// var fileSys utils.FileSys

	// stream, err := fileSys.NewFilenameStream(fileSys.GetWorkingDir())
	// if err != nil {
	// 	panic(err)
	// }

	// f, err := ff.FilterByLibEntry(stream, db.Anime())
	// if err != nil {
	// 	panic(err)
	// }

	// for _, file := range f {
	// 	fmt.Println(file.LibEntry.JPN_Title, file.Score)
	// 	fmt.Println(file.FileInfo.Filename)
	// }

	// 	fmt.Println("    Date:", entry.Date.Format("01/02/2006 3:04 PM"))
}
