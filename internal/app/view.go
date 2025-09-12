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
	b.WriteString(m.renderHeader() + "\n")

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

	// force panel to fill width
	pfw, _ := s.Panel.GetFrameSize()
	panelInner := max(20, m.width-pfw)
	b.WriteString(s.Panel.Width(panelInner).Render(content))

	// Footer
	if m.lastErr != nil {
		b.WriteString("\n" + s.Error.Render(fmt.Sprintf("ERR: %v", m.lastErr)))
	}
	help := "↑/k ↓/j move • Enter play • Space pause • n/p next/prev • Tab switch • Backspace up • q quit"
	b.WriteString("\n" + s.Footer.Render(fitTo(m.width, help)))

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

	pfw, _ := s.Panel.GetFrameSize()
	cw := max(20, m.width-pfw)
	rowPad := lipgloss.NewStyle().Width(cw)

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
		b.WriteString(rowPad.Render(rowStyle.Render(fitTo(cw, line))) + "\n")
	}
	if end < len(m.allSongs) {
		b.WriteString(rowPad.Render(s.ListRowDim.Render(fitTo(cw, fmt.Sprintf("  …and %d more", len(m.allSongs)-end)))))
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

	pfw, _ := s.Panel.GetFrameSize()
	cw := max(20, m.width-pfw)
	rows := m.maxRowsForList()

	// single column on narrow widths
	if m.width <= 90 {
		rowPad := lipgloss.NewStyle().Width(cw)
		switch m.level {
		case LevelArtist:
			if len(m.artists) == 0 {
				return s.ListRowDim.Render("(no artists)")
			}
			start, end := windowAroundCursor(m.cursor, rows, len(m.artists))
			for i := start; i < end; i++ {
				cur := "  "
				rowStyle := s.ListRow
				if i == m.cursor {
					cur = s.Cursor.Render("▍") + " "
					rowStyle = rowStyle.Bold(true)
				}
				b.WriteString(rowPad.Render(rowStyle.Render(fitTo(cw, cur+m.artists[i]))) + "\n")
			}
			return b.String()

		case LevelAlbum:
			if len(m.albums) == 0 {
				return s.ListRowDim.Render("(no albums)")
			}
			start, end := windowAroundCursor(m.cursor, rows, len(m.albums))
			for i := start; i < end; i++ {
				cur := "  "
				rowStyle := s.ListRow
				if i == m.cursor {
					cur = s.Cursor.Render("▍") + " "
					rowStyle = rowStyle.Bold(true)
				}
				b.WriteString(rowPad.Render(rowStyle.Render(fitTo(cw, cur+m.albums[i]))) + "\n")
			}
			return b.String()

		case LevelTrack:
			if len(m.tracks) == 0 {
				return s.ListRowDim.Render("(no tracks)")
			}
			start, end := windowAroundCursor(m.cursor, rows, len(m.tracks))
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
				b.WriteString(rowPad.Render(rowStyle.Render(fitTo(cw, line))) + "\n")
			}
			return b.String()
		}
		return b.String()
	}

	// two columns on wide widths
	leftW := (cw * 2) / 5
	rightW := cw - leftW - 2
	leftPad := lipgloss.NewStyle().Width(leftW)
	rightPad := lipgloss.NewStyle().Width(rightW)
	sep := "  "
	var left, right strings.Builder

	idx := buildIndexes(m.allSongs)

	switch m.level {
	case LevelArtist:
		if len(m.artists) == 0 {
			return s.ListRowDim.Render("(no artists)")
		}
		lstart, lend := windowAroundCursor(m.cursor, rows, len(m.artists))
		for i := lstart; i < lend; i++ {
			cur := "  "
			row := s.ListRow
			if i == m.cursor {
				cur = s.Cursor.Render("▍") + " "
				row = row.Bold(true)
			}
			left.WriteString(leftPad.Render(row.Render(fitTo(leftW, cur+m.artists[i]))) + "\n")
		}
		sel := m.selectArtist
		if sel == "" && len(m.artists) > 0 {
			sel = m.artists[m.cursor]
		}
		albums := idx.AlbumsByArtist[sel]
		if len(albums) == 0 {
			right.WriteString(rightPad.Render(s.ListRowDim.Render(fitTo(rightW, "(no albums)"))))
		} else {
			maxRows := min(rows, len(albums))
			for i := range maxRows {
				cur := "  "
				row := s.ListRow
				if m.selectAlbum != "" && albums[i] == m.selectAlbum {
					row = row.Bold(true)
				}
				right.WriteString(rightPad.Render(row.Render(fitTo(rightW, cur+albums[i]))) + "\n")
			}
		}

	case LevelAlbum:
		if len(m.albums) == 0 {
			return s.ListRowDim.Render("(no albums)")
		}
		lstart, lend := windowAroundCursor(m.cursor, rows, len(m.albums))
		for i := lstart; i < lend; i++ {
			cur := "  "
			row := s.ListRow
			if i == m.cursor {
				cur = s.Cursor.Render("▍") + " "
				row = row.Bold(true)
			}
			left.WriteString(leftPad.Render(row.Render(fitTo(leftW, cur+m.albums[i]))) + "\n")
		}
		artist := m.selectArtist
		album := m.selectAlbum
		if album == "" && len(m.albums) > 0 {
			album = m.albums[m.cursor]
		}
		trs := idx.TracksByArtistAlbum[keyAA(artist, album)]
		if len(trs) == 0 {
			right.WriteString(rightPad.Render(s.ListRowDim.Render(fitTo(rightW, "(no tracks)"))))
		} else {
			maxRows := min(rows, len(trs))
			for i := range maxRows {
				t := trs[i]
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
				right.WriteString(rightPad.Render(s.ListRow.Render(fitTo(rightW, prefix+title))) + "\n")
			}
		}

	case LevelTrack:
		if len(m.albums) == 0 {
			left.WriteString(leftPad.Render(s.ListRowDim.Render(fitTo(leftW, "(no albums)"))) + "\n")
		} else {
			maxRows := min(rows, len(m.albums))
			for i := range maxRows {
				cur := "  "
				row := s.ListRowDim
				if m.albums[i] == m.selectAlbum {
					row = s.ListRow
					cur = s.Cursor.Render("▍") + " "
				}
				left.WriteString(leftPad.Render(row.Render(fitTo(leftW, cur+m.albums[i]))) + "\n")
			}
		}
		if len(m.tracks) == 0 {
			right.WriteString(rightPad.Render(s.ListRowDim.Render(fitTo(rightW, "(no tracks)"))))
		} else {
			rstart, rend := windowAroundCursor(m.cursor, rows, len(m.tracks))
			for i := rstart; i < rend; i++ {
				t := m.tracks[i]
				cur := "  "
				row := s.ListRow
				if i == m.cursor {
					cur = s.Cursor.Render("▍") + " "
					row = row.Bold(true)
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
				right.WriteString(rightPad.Render(row.Render(fitTo(rightW, cur+prefix+title))) + "\n")
			}
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		left.String(),
		sep,
		right.String(),
	)
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

func (m Model) renderHeader() string {
	s := m.styles
	state := "disconnected"
	if m.connected {
		state = "connected"
	}
	title := s.AppTitle.Render("gompc")
	badge := s.HeaderBadge.Render(state)

	w := m.width
	nowFull := fmt.Sprintf("%s — %s [%s]", nz(m.now.Artist, "<unknown>"), nz(m.now.Title, "<untitled>"), nz(m.now.Album, "<unknown>"))
	nowCompact := fmt.Sprintf("%s — %s", nz(m.now.Artist, "<unknown>"), nz(m.now.Title, "<untitled>"))

	var nowShown string
	var barWidth int
	switch {
	case w > 0 && w < 50:
		nowShown = nowCompact
		barWidth = 8
	case w > 0 && w < 80:
		nowShown = nowCompact
		barWidth = 16
	default:
		nowShown = nowFull
		barWidth = clamp(w/3, 12, 30)
	}

	hfw, _ := s.Header.GetFrameSize()
	headerW := max(10, w-hfw)
	nowStr := s.HeaderNow.Render(fitTo(headerW/2, nowShown))

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		title,
		lipgloss.NewStyle().Foreground(colMuted).Render(" • MPD: "),
		badge,
		lipgloss.NewStyle().Render(" • "),
		nowStr,
	)

	out := s.Header.Width(headerW).Render(header)
	if m.now.Duration > 0 {
		out += "  " + m.renderProgress(barWidth)
	}
	return out
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

func (m Model) maxRowsForList() int {
	if m.height <= 0 {
		return 30
	}
	rows := m.height - 7
	if rows < 3 {
		rows = 3
	}
	return rows
}

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
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var _ tea.Model = (*Model)(nil)
