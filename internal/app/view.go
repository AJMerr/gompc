package app

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
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
	w := m.width
	nowShown := now
	barWidth := 24
	switch {
	case w > 0 && w < 50:
		nowShown = fmt.Sprintf("%s — %s", nz(m.now.Artist, "<unknown>"), nz(m.now.Title, "<untitled>"))
		barWidth = 8
	case w > 0 && w < 80:
		nowShown = fmt.Sprintf("%s — %s", nz(m.now.Artist, "<unknown>"), nz(m.now.Title, "<untitled>"))
		barWidth = 16
	default:
		barWidth = clamp(w/3, 12, 30)
	}
	nowStr := s.HeaderNow.Render(fitTo(max(10, w/2), nowShown)) // NEW: truncate if needed

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		title,
		lipgloss.NewStyle().Foreground(colMuted).Render(" • MPD: "),
		badge,
		lipgloss.NewStyle().Render(" • "),
		nowStr,
	)
	line := s.Header.Render(header)
	if m.now.Duration > 0 {
		line += "  " + m.renderProgress(barWidth) // width adapts with terminal
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
	b.WriteString("\n" + s.Footer.Render(fitTo(m.width, help))) // NEW

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

	rows := m.maxRowsForList()
	start, end := windowAroundCursor(m.cursor, rows, len(m.allSongs))

	cw := max(20, m.width-8)

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
		b.WriteString(rowStyle.Render(fitTo(cw, line)) + "\n") // NEW: truncate per-line
	}
	if end < len(m.allSongs) {
		b.WriteString(s.ListRowDim.Render(fitTo(cw, fmt.Sprintf("  …and %d more", len(m.allSongs)-end)))) // NEW
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

	cw := max(20, m.width-8)
	rows := m.maxRowsForList()

	switch m.level {
	case LevelArtist:
		if len(m.artists) == 0 {
			return s.ListRowDim.Render("(no artists)")
		}
		start, end := windowAroundCursor(m.cursor, rows, len(m.artists)) // NEW
		for i := start; i < end; i++ {
			cur := "  "
			rowStyle := s.ListRow
			if i == m.cursor {
				cur = s.Cursor.Render("▍") + " "
				rowStyle = rowStyle.Bold(true)
			}
			b.WriteString(rowStyle.Render(fitTo(cw, cur+m.artists[i])) + "\n") // NEW
		}

	case LevelAlbum:
		if len(m.albums) == 0 {
			return s.ListRowDim.Render("(no albums)")
		}
		start, end := windowAroundCursor(m.cursor, rows, len(m.albums)) // NEW
		for i := start; i < end; i++ {
			cur := "  "
			rowStyle := s.ListRow
			if i == m.cursor {
				cur = s.Cursor.Render("▍") + " "
				rowStyle = rowStyle.Bold(true)
			}
			b.WriteString(rowStyle.Render(fitTo(cw, cur+m.albums[i])) + "\n") // NEW
		}

	case LevelTrack:
		if len(m.tracks) == 0 {
			return s.ListRowDim.Render("(no tracks)")
		}
		start, end := windowAroundCursor(m.cursor, rows, len(m.tracks)) // NEW
		for i := start; i < end; i++ {
			t := m.tracks[i]
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
			line := cur + prefix + nz(t.Artist, "<unknown>") + " — " + title
			b.WriteString(rowStyle.Render(fitTo(cw, line)) + "\n") // NEW
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

// number of list rows that fit current height
func (m Model) maxRowsForList() int {
	if m.height <= 0 {
		return 30 // fallback before first WindowSizeMsg
	}
	rows := m.height - 7
	if rows < 3 {
		rows = 3
	}
	return rows
}

// center cursor inside a window of given size
func windowAroundCursor(cursor, rows, n int) (start, end int) {
	if n <= rows {
		return 0, n
	}
	start = cursor - rows/2
	if start < 0 {
		start = 0
	}
	end = start + rows
	if end > n {
		end = n
		start = end - rows
	}
	return
}

func fitTo(width int, s string) string {
	if width <= 0 {
		return s
	}
	return truncate.StringWithTail(s, uint(width), "…")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func clamp(x, lo, hi int) int {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

var _ tea.Model = (*Model)(nil)
