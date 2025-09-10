package app

import (
	"time"

	"github.com/AJMerr/gompc/internal/mpd"
)

// Connection lifecycle
type ConnectionMsg struct{ Conn mpd.Conn }
type ConnectionErrMsg struct{ Err error }

// Data
type LibLoadedMsg struct{ Tracks []mpd.Track }
type StatusMsg struct{ Now mpd.NowPlaying }

// Server Events
type IdleEventMsg struct{ Subs []string }

// UI timer tick
type TickMsg struct{ At time.Time }

type ErrMsg struct {
	Op  string
	Err error
}
