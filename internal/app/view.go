package app

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	s := m.styles
	var b strings.Builder

	// header
	state := "disconnected"
	if m.connected {
		state = "connected"
	}
	title := s.AppTitle.Render("gompc")
	badge := s.HeaderBadge.Render(state)

	now := fmt.Sprintf("%s — %s [%s]",
		nz(m.now.Artist, "<unknown>"),
		nz(m.now.Title, "<untitled>"),
		nz(m.now.Album, "<unknown>"),
	)
	nowStr := s.HeaderNow.Render(now)

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		title,
		lipgloss.NewStyle().Foreground(colMuted).Render(" • MPD: "),
		badge,
		lipgloss.NewStyle().Render(" • "),
		nowStr,
	)
	line := s.Header.Render(header)
	if m.now.Duration > 0 {
		line += "  " + m.renderProgress(24)
	}
	b.WriteString(line + "\n")

	// Tabs
	tabs := lipgloss.JoinHorizontal(lipgloss.Top,
		tabLabelStyled(s, m.tab == TabAll, "All"),
		tabLabelStyled(s, m.tab == TabArtists, "Artists"),
	)
	b.WriteString(tabs + "\n")

	// Content
	var content string
	switch m.tab {
	case TabAll:
		content = listAllViewStyled(m)
	case TabArtists:
		content = artistsViewStyled(m)
	}
	b.WriteString(s.Panel.Render(content))

	// Footer
	if m.lastErr != nil {
		b.WriteString("\n" + s.Error.Render(fmt.Sprintf("ERR: %v", m.lastErr)))
	}
	help := "↑/k ↓/j move • Enter play • Space pause • n/p next/prev • Tab switch • Backspace up • q quit"
	b.WriteString("\n" + s.Footer.Render(help))

	return b.String()
}

func tabLabelStyled(s Styles, active bool, label string) string {
	if active {
		return s.TabActive.Render(label)
	}
	return s.TabInactive.Render(label)
}

func listAllViewStyled(m Model) string {
	s := m.styles
	if len(m.allSongs) == 0 {
		return s.ListRowDim.Render("(no tracks)")
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
		cur := "  "
		rowStyle := s.ListRow
		if i == m.cursor {
			cur = s.Cursor.Render("▍") + " "
			rowStyle = rowStyle.Bold(true)
		}
		artist := nz(t.Artist, "<unknown>")
		title := t.Title
		if title == "" {
			title = baseNameFromURI(t.URI)
		}
		line := fmt.Sprintf("%s%s — %s [%s]", cur, artist, title, t.Album)
		b.WriteString(rowStyle.Render(line) + "\n")
	}
	if end < len(m.allSongs) {
		b.WriteString(s.ListRowDim.Render(fmt.Sprintf("  …and %d more", len(m.allSongs)-end)))
	}
	return b.String()
}

func artistsViewStyled(m Model) string {
	s := m.styles
	var b strings.Builder

	// breadcrumb
	if m.selectArtist == "" {
		b.WriteString(s.Breadcrumb.Render("Artists") + "\n")
	} else if m.selectAlbum == "" || m.level != LevelTrack {
		b.WriteString(s.Breadcrumb.Render("Artists › "+m.selectArtist) + "\n")
	} else {
		b.WriteString(s.Breadcrumb.Render("Artists › "+m.selectArtist+" › "+m.selectAlbum) + "\n")
	}

	switch m.level {
	case LevelArtist:
		if len(m.artists) == 0 {
			return s.ListRowDim.Render("(no artists)")
		}
		for i, a := range m.artists {
			cur := "  "
			rowStyle := s.ListRow
			if i == m.cursor {
				cur = s.Cursor.Render("▍") + " "
				rowStyle = rowStyle.Bold(true)
			}
			b.WriteString(rowStyle.Render(cur+a) + "\n")
		}

	case LevelAlbum:
		if len(m.albums) == 0 {
			return s.ListRowDim.Render("(no albums)")
		}
		for i, al := range m.albums {
			cur := "  "
			rowStyle := s.ListRow
			if i == m.cursor {
				cur = s.Cursor.Render("▍") + " "
				rowStyle = rowStyle.Bold(true)
			}
			b.WriteString(rowStyle.Render(cur+al) + "\n")
		}

	case LevelTrack:
		if len(m.tracks) == 0 {
			return s.ListRowDim.Render("(no tracks)")
		}
		for i, t := range m.tracks {
			cur := "  "
			rowStyle := s.ListRow
			if i == m.cursor {
				cur = s.Cursor.Render("▍") + " "
				rowStyle = rowStyle.Bold(true)
			}
			title := t.Title
			if title == "" {
				title = baseNameFromURI(t.URI)
			}
			prefix := ""
			if t.DiscNo > 0 || t.TrackNo > 0 {
				if t.DiscNo > 0 {
					prefix = fmt.Sprintf("[%d.%02d] ", t.DiscNo, t.TrackNo)
				} else {
					prefix = fmt.Sprintf("[%02d] ", t.TrackNo)
				}
				prefix = s.ListRowDim.Render(prefix)
			}
			b.WriteString(rowStyle.Render(cur+prefix+nz(t.Artist, "<unknown>")+" — "+title) + "\n")
		}
	}
	return b.String()
}

// Progress Bar
func (m Model) renderProgress(width int) string {
	if m.now.Duration <= 0 {
		return ""
	}
	p := float64(0)
	if m.now.Elapsed > 0 {
		p = float64(m.now.Elapsed) / float64(m.now.Duration)
		if p < 0 {
			p = 0
		}
		if p > 1 {
			p = 1
		}
	}
	fill := int(p * float64(width))
	filled := strings.Repeat("█", fill)
	rest := strings.Repeat(" ", width-fill)
	bar := m.styles.ProgressFill.Render(filled) + rest
	return m.styles.ProgressOuter.Render(bar) + " " + m.styles.HeaderNow.Render(
		fmt.Sprintf("%s/%s", truncDur(m.now.Elapsed), truncDur(m.now.Duration)),
	)
}

func truncDur(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	return d.Truncate(time.Second).String()
}

func baseNameFromURI(uri string) string {
	if uri == "" {
		return "<untitled>"
	}
	slash := strings.LastIndex(uri, "/")
	if slash == -1 || slash == len(uri)-1 {
		return uri
	}
	return uri[slash+1:]
}

var _ tea.Model = (*Model)(nil)
