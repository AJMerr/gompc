package mpd

import (
	"context"
	"time"
)

type Config struct {
	Host    string
	Port    int
	Timeout time.Duration
}

type Track struct {
	URI      string
	Title    string
	Artist   string
	Album    string
	Duration time.Duration
}

type NowPlaying struct {
	Title    string
	Artist   string
	Album    string
	Elapsed  time.Duration
	Duration time.Duration
	Playing  bool
}

// Produces a connection for reconnecting
type Client interface {
	Connect(ctx context.Context, cfg Config) (Conn, error)
}

type Conn interface {
	Close() error

	// Library
	ListAll(ctx context.Context) ([]Track, error)

	// Playback controls
	Play(ctx context.Context, uri string) error
	TogglePause(ctx context.Context) error
	Next(ctx context.Context) error
	Prev(ctx context.Context) error

	// Status
	Status(ctx context.Context) (NowPlaying, error)

	Idle(ctx context.Context, subs []string) ([]string, error)
}

func NewClient() Client {
	return nil
}
