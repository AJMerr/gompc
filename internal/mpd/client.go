package mpd

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
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
	TrackNo  int
	DiscNo   int
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

	QueueClear(ctx context.Context) error
	QueueAdd(ctx context.Context, uri string) error
	QueueAddID(ctx context.Context, uri string) error
	PlayPos(ctx context.Context, pos int) error
	PlayID(ctx context.Context, id int) error
}

var _ Client = (*client)(nil)

type client struct {
	defaultTimeout time.Duration
}

func NewClient() Client {
	return &client{defaultTimeout: 3 * time.Second}
}

func (c *client) Connect(ctx context.Context, cfg Config) (Conn, error) {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = c.defaultTimeout
	}
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))

	d := &net.Dialer{Timeout: timeout}
	nc, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	br := bufio.NewReader(nc)
	_ = nc.SetReadDeadline(time.Now().Add(timeout))
	hello, err := br.ReadString('\n')
	if err != nil {
		_ = nc.Close()
		return nil, err
	}
	hello = strings.TrimSpace(hello)
	if !strings.HasPrefix(hello, "OK MPD ") {
		_ = nc.Close()
		return nil, fmt.Errorf("unexpected greeting: %q", hello)
	}

	return &tcpConn{
		conn:    nc,
		rd:      br,
		timeout: timeout,
	}, nil
}

type tcpConn struct {
	conn    net.Conn
	rd      *bufio.Reader
	timeout time.Duration
	mu      sync.Mutex
}

var _ Conn = (*tcpConn)(nil)

func (t *tcpConn) Close() error { return t.conn.Close() }

func (t *tcpConn) cmd(ctx context.Context, line string) ([]string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	deadline := time.Now().Add(t.timeout)
	_ = t.conn.SetWriteDeadline(deadline)
	if _, err := t.conn.Write([]byte(line + "\n")); err != nil {
		return nil, err
	}

	_ = t.conn.SetReadDeadline(deadline)
	var out []string
	for {
		s, err := t.rd.ReadString('\n')
		if err != nil {
			return nil, err
		}
		s = strings.TrimRight(s, "\r\n")
		if s == "OK" {
			return out, nil
		}
		if strings.HasPrefix(s, "ACK ") {
			return nil, fmt.Errorf(s)
		}
		out = append(out, s)
	}
}

func (t *tcpConn) ListAll(ctx context.Context) ([]Track, error) {
	lines, err := t.cmd(ctx, "listallinfo")
	if err != nil {
		return nil, err
	}
	var tracks []Track
	var cur *Track

	flush := func() {
		if cur != nil {
			tracks = append(tracks, *cur)
			cur = nil
		}
	}

	for _, ln := range lines {
		switch {
		case strings.HasPrefix(ln, "file: "):
			flush()
			cur = &Track{URI: strings.TrimPrefix(ln, "file: ")}
		case cur != nil && strings.HasPrefix(ln, "Title: "):
			cur.Title = strings.TrimPrefix(ln, "Title: ")
		case cur != nil && strings.HasPrefix(ln, "Artist: "):
			cur.Artist = strings.TrimPrefix(ln, "Artist: ")
		case cur != nil && strings.HasPrefix(ln, "Album: "):
			cur.Album = strings.TrimPrefix(ln, "Album: ")
		case cur != nil && strings.HasPrefix(ln, "Time: "):
			secs := strings.TrimPrefix(ln, "Time: ")
			if d, ok := parseSecs(secs); ok {
				cur.Duration = d
			}
			// ignore directory/playlist/etc lines
		case cur != nil && strings.HasPrefix(ln, "Track: "):
			cur.TrackNo = parseTrackNum(strings.TrimPrefix(ln, "Track: "))
		case cur != nil && strings.HasPrefix(ln, "Disc: "):
			cur.DiscNo = parseIntSafe(strings.TrimPrefix(ln, "Disc: "))
		}
	}
	flush()
	return tracks, nil
}

func parseIntSafe(s string) int {
	s = strings.TrimSpace(s)
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	if i := strings.IndexByte(s, '/'); i > 0 {
		if n, err := strconv.Atoi(s[:i]); err == nil {
			return n
		}
	}
	return 0
}

func parseTrackNum(s string) int {
	return parseIntSafe(s)
}

func (t *tcpConn) Play(ctx context.Context, uri string) error {
	if _, err := t.cmd(ctx, "clear"); err != nil {
		return err
	}
	if _, err := t.cmd(ctx, `add "`+escape(uri)+`"`); err != nil {
		return err
	}
	_, err := t.cmd(ctx, "play")
	return err
}

func (t *tcpConn) TogglePause(ctx context.Context) error {
	// Check current state
	lines, err := t.cmd(ctx, "status")
	if err != nil {
		return err
	}
	state := ""
	for _, ln := range lines {
		if strings.HasPrefix(ln, "state: ") {
			state = strings.TrimPrefix(ln, "state: ")
			break
		}
	}

	switch state {
	case "play":
		_, err = t.cmd(ctx, "pause 1") // pause
		return err
	case "pause":
		_, err = t.cmd(ctx, "pause 0") // resume
		return err
	case "stop":
		_, err = t.cmd(ctx, "play") // start playback if stopped
		return err
	default:
		// Fallback: try protocol toggle
		_, err = t.cmd(ctx, "pause")
		return err
	}
}

func (t *tcpConn) Next(ctx context.Context) error {
	_, err := t.cmd(ctx, "next")
	return err
}

func (t *tcpConn) Prev(ctx context.Context) error {
	_, err := t.cmd(ctx, "previous")
	return err
}

func (t *tcpConn) Status(ctx context.Context) (NowPlaying, error) {
	stLines, err := t.cmd(ctx, "status")
	if err != nil {
		return NowPlaying{}, err
	}
	m := kvLower(stLines)

	var np NowPlaying
	np.Playing = m["state"] == "play"

	// Prefer precise fields if present
	if v, ok := m["elapsed"]; ok {
		if d, ok2 := parseSecs(v); ok2 {
			np.Elapsed = d
		}
	}
	if v, ok := m["duration"]; ok {
		if d, ok2 := parseSecs(v); ok2 {
			np.Duration = d
		}
	} else if v, ok := m["time"]; ok { // fallback "elapsed:total"
		if e, d, ok2 := parseTimePair(v); ok2 {
			np.Elapsed, np.Duration = e, d
		}
	}

	// Merge metadata from currentsong (best-effort)
	if csLines, err := t.cmd(ctx, "currentsong"); err == nil {
		cs := kvLower(csLines)
		np.Title = cs["title"]
		np.Artist = cs["artist"]
		np.Album = cs["album"]
	}

	return np, nil
}

func (t *tcpConn) Idle(ctx context.Context, subs []string) ([]string, error) {
	cmd := "idle"
	if len(subs) > 0 {
		cmd += " " + strings.Join(subs, " ")
	}
	lines, err := t.cmd(ctx, cmd)
	if err != nil {
		// Treat timeouts as "no events yet" so callers can re-issue Idle.
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			return nil, nil
		}
		return nil, err
	}

	var out []string
	for _, ln := range lines {
		if strings.HasPrefix(ln, "changed: ") {
			out = append(out, strings.TrimPrefix(ln, "changed: "))
		}
	}
	return out, nil
}

func (t *tcpConn) QueueClear(ctx context.Context) error {
	_, err := t.cmd(ctx, "clear")
	return err
}
func (t *tcpConn) QueueAdd(ctx context.Context, uri string) error {
	_, err := t.cmd(ctx, `add "`+escape(uri)+`"`)
	return err
}
func (t *tcpConn) QueueAddID(ctx context.Context, uri string) (int, error) {
	lines, err := t.cmd(ctx, `addid "`+escape(uri)+`"`)
	if err != nil {
		return 0, err
	}
	for _, ln := range lines {
		if strings.HasPrefix(ln, "Id: ") {
			n, _ := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(ln, "Id: ")))
			if n > 0 {
				return n, nil
			}
		}
	}
	return 0, fmt.Errorf("addid: missing Id")
}
func (t *tcpConn) PlayPos(ctx context.Context, pos int) error {
	_, err := t.cmd(ctx, fmt.Sprintf("play %d", pos))
	return err
}
func (t *tcpConn) PlayID(ctx context.Context, id int) error {
	_, err := t.cmd(ctx, fmt.Sprintf("playid %d", id))
	return err
}

func escape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

func kvLower(lines []string) map[string]string {
	m := make(map[string]string, len(lines))
	for _, ln := range lines {
		if i := strings.Index(ln, ": "); i >= 0 {
			k := strings.ToLower(ln[:i])
			v := strings.TrimSpace(ln[i+2:])
			m[k] = v
		}
	}
	return m
}

func parseSecs(s string) (time.Duration, bool) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return time.Duration(f * float64(time.Second)), true
}

func parseTimePair(s string) (elapsed, total time.Duration, ok bool) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	e, ok1 := parseSecs(parts[0])
	t, ok2 := parseSecs(parts[1])
	return e, t, ok1 && ok2
}
