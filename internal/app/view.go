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
	if len(m.tracks) == 0 {
		return "\n(no tracks)\n"
	}

	const maxRows = 30
	start := m.cursor - maxRows/2
	if start < 0 {
		start = 0
	}
	end := start + maxRows
	if end > len(m.tracks) {
		end = len(m.tracks)
	}
	if m.cursor >= end {
		start = m.cursor - maxRows + 1
		if start < 0 {
			start = 0
		}
		end = start + maxRows
		if end > len(m.tracks) {
			end = len(m.tracks)
		}
	}

	var b strings.Builder
	for i := start; i < end; i++ {
		t := m.tracks[i]
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
	if end < len(m.tracks) {
		fmt.Fprintf(&b, "  …and %d more\n", len(m.tracks)-end)
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
	// TODO: render artist/album/track levels based on m.level
	return "\n(artists/albums/tracks here)\n"
}

// Make sure Model implements tea.Model
var _ tea.Model = (*Model)(nil)
