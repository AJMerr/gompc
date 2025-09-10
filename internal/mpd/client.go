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
	return nil, fmt.Errorf("TODO: implement ListAll")
}

func (t *tcpConn) Play(ctx context.Context, uri string) error {
	return fmt.Errorf("TODO: implement Play")
}

func (t *tcpConn) TogglePause(ctx context.Context) error {
	return fmt.Errorf("TODO: implement TogglePause")
}

func (t *tcpConn) Next(ctx context.Context) error {
	return fmt.Errorf("TODO: implement Next")
}

func (t *tcpConn) Prev(ctx context.Context) error {
	return fmt.Errorf("TODO: implement Prev")
}

func (t *tcpConn) Status(ctx context.Context) (NowPlaying, error) {
	return NowPlaying{}, fmt.Errorf("TODO: implement Status")
}

func (t *tcpConn) Idle(ctx context.Context, subs []string) ([]string, error) {
	return nil, fmt.Errorf("TODO: implement Idle")
}

func escape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
