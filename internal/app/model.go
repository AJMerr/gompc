package app

import (
	"github.com/AJMerr/gompc/internal/mpd"
	tea "github.com/charmbracelet/bubbletea"
)

// Tabs
type Tab int

const (
	TabAll Tab = iota
	TabArtists
)

type Keymap struct {
	Up, Down     string
	Tab          string
	Enter, Space string
	Next, Prev   string
	Back         string
	Quit         string
}

// Heirarchy state for Artists/Albums
type Level int

const (
	LevelArtist Level = iota
	LevelAlbum
	LevelTrack
)

type Model struct {
	deps Deps
	conn mpd.Conn

	// UI state
	tab       Tab
	level     Level
	cursor    int
	loading   bool
	lastErr   error
	connected bool

	// Indexes
	allSongs []mpd.Track
	artists  []string
	albums   []string
	tracks   []mpd.Track
	now      mpd.NowPlaying

	// Selections
	selectArtist string
	selectAlbum  string

	keys Keymap
}

func New(d Deps) Model {
	return Model{
		deps: d,
		tab:  TabAll,
		keys: Keymap{
			Up: "up/k", Down: "down/j", Tab: "tab",
			Enter: "enter", Space: "space",
			Next: "n", Prev: "p", Back: "backspace",
			Quit: "q",
		},
		loading: true,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		ConnectCmd(m.deps),
		TickCmd(m.deps.Cfg.Timeout/4),
	)
}
