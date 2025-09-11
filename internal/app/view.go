package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) View() string {
	var b strings.Builder

	// Header
	state := "disconnected"
	if m.connected {
		state = "connected"
	}
	np := fmt.Sprintf("%s — %s [%s]", m.now.Artist, m.now.Title, m.now.Album)
	fmt.Fprintf(&b, "gompc  •  MPD: %s  •  Now Playing: %s\n", state, np)
	fmt.Fprintf(&b, "Tabs: %s | %s\n", tabLabel(m.tab == TabAll, "All"), tabLabel(m.tab == TabArtists, "Artists"))

	// Content
	switch m.tab {
	case TabAll:
		b.WriteString(listAllView(m))
	case TabArtists:
		b.WriteString(artistsView(m))
	}

	// Footer/help
	if m.lastErr != nil {
		fmt.Fprintf(&b, "\nERR: %v\n", m.lastErr)
	}
	b.WriteString("\n↑/k ↓/j move • Enter play • Space pause • n/p next/prev • Tab switch • q quit\n")
	return b.String()
}

func tabLabel(active bool, s string) string {
	if active {
		return "[" + s + "]"
	}
	return s
}

func listAllView(m Model) string {
	if len(m.allSongs) == 0 {
		return "\n(no tracks)\n"
	}

	const maxRows = 30
	start := m.cursor - maxRows/2
	if start < 0 {
		start = 0
	}
	end := start + maxRows
	if end > len(m.allSongs) {
		end = len(m.allSongs)
	}
	if m.cursor >= end {
		start = m.cursor - maxRows + 1
		if start < 0 {
			start = 0
		}
		end = start + maxRows
		if end > len(m.allSongs) {
			end = len(m.allSongs)
		}
	}

	var b strings.Builder
	for i := start; i < end; i++ {
		t := m.allSongs[i]
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		artist := t.Artist
		if artist == "" {
			artist = "<unknown>"
		}
		title := t.Title
		if title == "" {
			// fall back to file name if no tag
			title = baseNameFromURI(t.URI)
		}
		fmt.Fprintf(&b, "%s%s — %s [%s]\n", cursor, artist, title, t.Album)
	}
	if end < len(m.allSongs) {
		fmt.Fprintf(&b, "  …and %d more\n", len(m.allSongs)-end)
	}
	return b.String()
}

func baseNameFromURI(uri string) string {
	// URIs from MPD use forward slashes; avoid OS-specific filepath here.
	if uri == "" {
		return "<untitled>"
	}
	slash := strings.LastIndex(uri, "/")
	if slash == -1 || slash == len(uri)-1 {
		return uri
	}
	return uri[slash+1:]
}

func artistsView(m Model) string {
	var b strings.Builder
	// breadcrumb
	if m.selectArtist == "" {
		b.WriteString("\nArtists\n")
	} else if m.selectAlbum == "" || m.level != LevelTrack {
		fmt.Fprintf(&b, "\nArtists › %s\n", m.selectArtist)
	} else {
		fmt.Fprintf(&b, "\nArtists › %s › %s\n", m.selectArtist, m.selectAlbum)
	}

	switch m.level {
	case LevelArtist:
		if len(m.artists) == 0 {
			b.WriteString("(no artists)\n")
			return b.String()
		}
		for i, a := range m.artists {
			cur := "  "
			if i == m.cursor {
				cur = "> "
			}
			fmt.Fprintf(&b, "%s%s\n", cur, a)
		}
	case LevelAlbum:
		if len(m.albums) == 0 {
			b.WriteString("(no albums)\n")
			return b.String()
		}
		for i, al := range m.albums {
			cur := "  "
			if i == m.cursor {
				cur = "> "
			}
			fmt.Fprintf(&b, "%s%s\n", cur, al)
		}
	case LevelTrack:
		if len(m.tracks) == 0 {
			b.WriteString("(no tracks)\n")
			return b.String()
		}
		for i, t := range m.tracks {
			cur := "  "
			if i == m.cursor {
				cur = "> "
			}
			title := t.Title
			if title == "" {
				title = baseNameFromURI(t.URI)
			}
			fmt.Fprintf(&b, "%s%s — %s\n", cur, nz(t.Artist, "<unknown>"), title)
		}
	}
	return b.String()
}

var _ tea.Model = (*Model)(nil)
