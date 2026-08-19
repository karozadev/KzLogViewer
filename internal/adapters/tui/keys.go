package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap centralizes every keybinding recognized by the TUI, doubling as
// the source of the help panel text.
type keyMap struct {
	Up            key.Binding
	Down          key.Binding
	Top           key.Binding
	Bottom        key.Binding
	Enter         key.Binding
	Escape        key.Binding
	Search        key.Binding
	CycleMode     key.Binding
	NextContainer key.Binding
	TogglePause   key.Binding
	Quit          key.Binding
	Help          key.Binding
}

var keys = keyMap{
	Up:            key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "move up")),
	Down:          key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "move down")),
	Top:           key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
	Bottom:        key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom / follow")),
	Enter:         key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "toggle detail")),
	Escape:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Search:        key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
	CycleMode:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "cycle search mode")),
	NextContainer: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "filter container")),
	TogglePause:   key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pause/resume")),
	Quit:          key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Help:          key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
}
