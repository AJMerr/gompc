package app

import (
	"context"
	"time"

	"github.com/AJMerr/gompc/internal/mpd"
	tea "github.com/charmbracelet/bubbletea"
)

// Dependencies passed into command constructors:
type Deps struct {
	Client mpd.Client // your mpd.NewClient()
	Cfg    mpd.Config // resolved host/port/timeout
}

// Connect to MPD and emit ConnectedMsg or ConnectErrMsg.
func ConnectCmd(d Deps) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), d.Cfg.Timeout)
		defer cancel()
		c, err := d.Client.Connect(ctx, d.Cfg)
		if err != nil {
			return ConnectionErrMsg{Err: err}
		}
		return ConnectionMsg{Conn: c}
	}
}

// Fetch the full library and emit LibraryLoadedMsg or ErrMsg{Op:"library"}.
func FetchLibraryCmd(conn mpd.Conn) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		tracks, err := conn.ListAll(ctx)
		if err != nil {
			return ErrMsg{Op: "library", Err: err}
		}
		return LibLoadedMsg{Tracks: tracks}
	}
}

// Ask MPD for current status and emit StatusMsg or ErrMsg{Op:"status"}.
func StatusCmd(conn mpd.Conn) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		now, err := conn.Status(ctx)
		if err != nil {
			return ErrMsg{Op: "status", Err: err}
		}
		return StatusMsg{Now: now}
	}
}

// Send a playback action (play/toggle/next/prev) then re-fetch Status.
type PlayAction int

const (
	ActionPlayURI PlayAction = iota
	ActionTogglePause
	ActionNext
	ActionPrev
)

type PlayRequest struct {
	Action PlayAction
	URI    string // used when ActionPlayURI
}

func PlaybackCmd(conn mpd.Conn, req PlayRequest) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		switch req.Action {
		case ActionPlayURI:
			if err := conn.Play(ctx, req.URI); err != nil {
				return ErrMsg{Op: "play", Err: err}
			}
		case ActionTogglePause:
			if err := conn.TogglePause(ctx); err != nil {
				return ErrMsg{Op: "pause", Err: err}
			}
		}
		now, err := conn.Status(ctx)
		if err != nil {
			return ErrMsg{Op: "status", Err: err}
		}
		return StatusMsg{Now: now}
	}
}

// Long-poll MPD idle for player/database changes.
func IdleCmd(conn mpd.Conn, subs []string) tea.Cmd {
	return func() tea.Msg {
		// TODO: evs, err := conn.Idle(ctx, subs)
		// return IdleEventMsg{Subs: evs} or ErrMsg{Op:"idle"}
		return ErrMsg{Op: "idle", Err: nil} // placeholder
	}
}

// UI tick for animating elapsed time; re-schedule from Update.
func TickCmd(interval time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(interval)
		return TickMsg{At: time.Now()}
	}
}

func EnqueueAndPlayCmd(conn mpd.Conn, uris []string, start int) tea.Cmd {
	// slice from start
	if start < 0 {
		start = 0
	}
	if start >= len(uris) {
		return func() tea.Msg { return ErrMsg{Op: "enqueue", Err: nil} }
	}
	u := uris[start:]

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := conn.QueueClear(ctx); err != nil {
			return ErrMsg{Op: "clear", Err: err}
		}
		// first track: addid to get ID, then add the rest
		firstID := 0
		var err error
		if firstID, err = conn.QueueAddID(ctx, u[0]); err != nil {
			return ErrMsg{Op: "addid", Err: err}
		}
		for _, uri := range u[1:] {
			if uri == "" {
				continue
			}
			if err := conn.QueueAdd(ctx, uri); err != nil {
				return ErrMsg{Op: "add", Err: err}
			}
		}
		if err := conn.PlayID(ctx, firstID); err != nil {
			// fallback to position if needed
			_ = conn.PlayPos(ctx, 0)
		}
		now, err := conn.Status(ctx)
		if err != nil {
			return ErrMsg{Op: "status", Err: err}
		}
		return StatusMsg{Now: now}
	}
}

func EnqueueAllFromCursor(conn mpd.Conn, tracks []mpd.Track, start int) tea.Cmd {
	uris := make([]string, len(tracks))
	for i := range tracks {
		uris[i] = tracks[i].URI
	}
	return EnqueueAndPlayCmd(conn, uris, start)
}

func EnqueueAlbumFromCursor(conn mpd.Conn, tracks []mpd.Track, start int) tea.Cmd {
	uris := make([]string, len(tracks))
	for i := range tracks {
		uris[i] = tracks[i].URI
	}
	return EnqueueAndPlayCmd(conn, uris, start)
}
