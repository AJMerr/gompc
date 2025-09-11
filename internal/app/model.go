package app

import (
	"sort"
	"strings"

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
		deps:  d,
		tab:   TabAll,
		level: LevelArtist,
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

func nz(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

func keyAA(artist, album string) string {
	// normalized key for (artist,album)
	return strings.ToLower(nz(artist, "<unknown>")) + "\x00" + strings.ToLower(nz(album, "<unknown>"))
}

type libIndex struct {
	Artists             []string
	AlbumsByArtist      map[string][]string
	TracksByArtistAlbum map[string][]mpd.Track // keyAA
}

func buildIndexes(ts []mpd.Track) libIndex {
	artistsSet := map[string]struct{}{}
	albumsByArtist := map[string]map[string]struct{}{}
	tracksByAA := map[string][]mpd.Track{}

	for _, t := range ts {
		a := nz(t.Artist, "<unknown>")
		al := nz(t.Album, "<unknown>")
		artistsSet[a] = struct{}{}

		if _, ok := albumsByArtist[a]; !ok {
			albumsByArtist[a] = map[string]struct{}{}
		}
		albumsByArtist[a][al] = struct{}{}

		k := keyAA(a, al)
		tracksByAA[k] = append(tracksByAA[k], t)
	}

	// materialize + sort
	var artists []string
	for a := range artistsSet {
		artists = append(artists, a)
	}
	sort.Strings(artists)

	albumsOut := make(map[string][]string, len(albumsByArtist))
	for a, set := range albumsByArtist {
		var list []string
		for al := range set {
			list = append(list, al)
		}
		sort.Strings(list)
		albumsOut[a] = list
	}

	// Sorts track by track number
	for k := range tracksByAA {
		sort.SliceStable(tracksByAA[k], func(i, j int) bool {
			ti, tj := tracksByAA[k][i], tracksByAA[k][j]
			// Disc first (0 = unknown -> push to end)
			di, dj := ti.DiscNo, tj.DiscNo
			if di != dj {
				if di == 0 || dj == 0 {
					return dj == 0 // known discs come before unknown
				}
				return di < dj
			}
			// Then track number (0 = unknown -> push to end)
			if ti.TrackNo != tj.TrackNo {
				if ti.TrackNo == 0 || tj.TrackNo == 0 {
					return tj.TrackNo == 0 // known tracks before unknown
				}
				return ti.TrackNo < tj.TrackNo
			}
			// Stable tie-breakers
			if ti.Title != tj.Title {
				return ti.Title < tj.Title
			}
			return ti.URI < tj.URI
		})
	}

	return libIndex{
		Artists:             artists,
		AlbumsByArtist:      albumsOut,
		TracksByArtistAlbum: tracksByAA,
	}
}
