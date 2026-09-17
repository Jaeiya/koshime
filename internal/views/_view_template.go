
type SomeModel struct {
	windowSize tea.WindowSizeMsg
	ui         struct {
		loader ui.LoaderModel
	}
	err error
}

func newSomeModel() SomeModel {
	m := SomeModel{}
	m.ui.loader = ui.NewLoader()
}

func (m SomeModel) Init() tea.Cmd {
	return nil
}

func (m SomeModel) Update(msg tea.Msg) (SomeModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, ui.KeyMap.MainMenu):
			return m, exitToMenu
		}

	case error:
		m.err = msg
	}

	return m, tea.Batch(cmds...)
}

func (m SomeModel) View() tea.View {
	v := tea.NewView("")

	if m.loader.IsLoading() {
		v.Content = ui.Style.MarginTop(1).Render(m.loader.View())
		return v
	}

	if m.err != nil {
		v.Content = ui.DisplayError(m.err)
		return v
	}

	return v
}

func (m SomeModel) ShortHelp() []key.Binding {
	return []key.Binding{}
}

func (m SomeModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}
