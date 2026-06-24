package ui

import "charm.land/bubbles/v2/key"

// keyMap holds every binding used across the app and its overlays.
type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Top      key.Binding
	Bottom   key.Binding

	Tab    key.Binding
	Enter  key.Binding
	Mark   key.Binding
	Search key.Binding

	Branches  key.Binding
	Worktrees key.Binding
	Checkout  key.Binding
	Add       key.Binding
	Delete    key.Binding

	Refresh key.Binding
	Help    key.Binding
	Esc     key.Binding
	Quit    key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		PageUp:   key.NewBinding(key.WithKeys("pgup", "ctrl+u"), key.WithHelp("pgup", "page up")),
		PageDown: key.NewBinding(key.WithKeys("pgdown", "ctrl+d"), key.WithHelp("pgdn", "page down")),
		Top:      key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
		Bottom:   key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),

		Tab:    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch pane")),
		Enter:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "details")),
		Mark:   key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "mark A/B")),
		Search: key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),

		Branches:  key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "branches")),
		Worktrees: key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "worktrees")),
		Checkout:  key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "checkout")),
		Add:       key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		Delete:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "remove")),

		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Esc:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

// ShortHelp implements help.KeyMap.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Tab, k.Enter, k.Search, k.Mark, k.Branches, k.Worktrees, k.Help, k.Quit}
}

// FullHelp implements help.KeyMap.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown, k.Top, k.Bottom},
		{k.Tab, k.Enter, k.Mark, k.Search, k.Refresh},
		{k.Branches, k.Worktrees, k.Checkout, k.Add, k.Delete},
		{k.Help, k.Esc, k.Quit},
	}
}
