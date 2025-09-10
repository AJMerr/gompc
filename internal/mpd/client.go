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
