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
		m.allSongs = msg.Tracks

		idx := buildIndexes(m.allSongs)
		m.artists = idx.Artists
		m.albums = nil
		m.tracks = nil
		m.selectArtist, m.selectAlbum = "", ""
		m.level = LevelArtist
		if m.tab == TabArtists {
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

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

		// Key handling (add your preferred key lib later)
	case tea.KeyMsg:
		// Spacebar
		if msg.Type == tea.KeySpace {
			if m.conn != nil {
				return m, PlaybackCmd(m.conn, PlayRequest{Action: ActionTogglePause})
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "tab":
			if m.tab == TabAll {
				m.tab = TabArtists
				m.level = LevelArtist
				m.cursor = 0
			} else {
				m.tab = TabAll
				m.cursor = 0
			}
			return m, nil

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case "down", "j":
			limit := 0
			if m.tab == TabAll {
				limit = len(m.allSongs)
			} else { // artists tab
				switch m.level {
				case LevelArtist:
					limit = len(m.artists)
				case LevelAlbum:
					limit = len(m.albums)
				case LevelTrack:
					limit = len(m.tracks)
				}
			}
			if m.cursor+1 < limit {
				m.cursor++
			}
			return m, nil

		case "backspace", "h":
			if m.tab == TabArtists {
				switch m.level {
				case LevelTrack:
					m.level = LevelAlbum
					m.cursor = 0
					m.tracks = nil
				case LevelAlbum:
					m.level = LevelArtist
					m.cursor = 0
					m.albums = nil
					m.selectAlbum = ""
				case LevelArtist:
					// stay
				}
			}
			return m, nil

		case "enter":
			if m.conn == nil {
				return m, nil
			}
			if m.tab == TabAll {
				if len(m.allSongs) == 0 {
					return m, nil
				}
				return m, EnqueueAllFromCursor(m.conn, m.allSongs, m.cursor)
			}
			// Artists tab
			switch m.level {
			case LevelArtist:
				if len(m.artists) == 0 {
					return m, nil
				}
				m.selectArtist = m.artists[m.cursor]
				idx := buildIndexes(m.allSongs)
				m.albums = idx.AlbumsByArtist[m.selectArtist]
				m.level = LevelAlbum
				m.cursor = 0
				return m, nil

			case LevelAlbum:
				if len(m.albums) == 0 {
					return m, nil
				}
				m.selectAlbum = m.albums[m.cursor]
				idx := buildIndexes(m.allSongs)
				m.tracks = idx.TracksByArtistAlbum[keyAA(m.selectArtist, m.selectAlbum)]
				m.level = LevelTrack
				m.cursor = 0
				return m, nil

			case LevelTrack:
				if len(m.tracks) == 0 {
					return m, nil
				}
				return m, EnqueueAlbumFromCursor(m.conn, m.tracks, m.cursor)
			}
		}
	}
	return m, nil
}
