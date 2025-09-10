package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case ConnectionMsg:
		m.conn = msg.Conn
		m.connected = true
		m.loading = true
		return m, tea.Batch(
			FetchLibraryCmd(m.conn),
			StatusCmd(m.conn),
			IdleCmd(m.conn, []string{"player", "database"}),
		)

	case ConnectionErrMsg:
		m.lastErr = msg.Err
		m.connected = false
		m.loading = false
		return m, nil

	case LibLoadedMsg:
		m.loading = false
		m.tracks = msg.Tracks
		if m.cursor >= len(m.tracks) {
			m.cursor = 0
		}
		return m, nil

	case StatusMsg:
		m.now = msg.Now
		return m, nil

	case IdleEventMsg:
		// React to server events; always resubscribe
		// - if "player" in Subs: refresh status
		// - if "database" in Subs: refetch library
		return m, tea.Batch(
			StatusCmd(m.conn),       // decide conditionally in your impl
			FetchLibraryCmd(m.conn), // decide conditionally in your impl
			IdleCmd(m.conn, []string{"player", "database"}),
		)

	case TickMsg:
		if m.now.Playing {
			m.now.Elapsed += 500_000_000
		}
		return m, TickCmd(500_000_000) // 500ms

	case ErrMsg:
		m.lastErr = msg.Err
		return m, nil

	// Key handling (add your preferred key lib later)
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "tab":
			if m.tab == TabAll {
				m.tab = TabArtists
			} else {
				m.tab = TabAll
			}
			m.cursor = 0
			return m, nil
		case "enter":
			if len(m.tracks) == 0 || m.conn == nil {
				return m, nil
			}
			uri := m.tracks[m.cursor].URI
			return m, PlaybackCmd(m.conn, PlayRequest{Action: ActionPlayURI, URI: uri})
		case " ", "space":
			if m.conn == nil {
				return m, nil
			}
			return m, PlaybackCmd(m.conn, PlayRequest{Action: ActionTogglePause})
		case "n":
			// TODO: next
			return m, nil
		case "p":
			// TODO: prev
			return m, nil
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "j":
			if m.cursor+1 < len(m.tracks) {
				m.cursor++
			}
			return m, nil
		case "backspace":
			// TODO: go up one level in Artists tab
			return m, nil
		}
	}

	return m, nil
}
