package doctor

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

type Config struct {
	Host      string
	Port      int
	TimeoutMS int
}

type mpdConn struct {
	conn   net.Conn
	reader *bufio.Reader
}

func (m *mpdConn) Close() error { return m.conn.Close() }

func dial(ctx context.Context, addr string, timeout time.Duration) (*mpdConn, string, time.Duration, error) {
	start := time.Now()

	d := &net.Dialer{Timeout: timeout}
	c, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, "", time.Since(start), err
	}
	r := bufio.NewReader(c)
	_ = c.SetReadDeadline(time.Now().Add(timeout))
	greet, err := r.ReadString('\n')
	if err != nil {
		c.Close()
		return nil, "", time.Since(start), err
	}
	greet = strings.TrimSpace(greet)
	if !strings.HasPrefix(greet, "OK MPD ") {
		c.Close()
		return nil, "", time.Since(start), fmt.Errorf("unexpected greeting: %q", greet)
	}
	version := strings.TrimPrefix(greet, "OK MPD ")
	return &mpdConn{conn: c, reader: r}, version, time.Since(start), nil
}

func (m *mpdConn) cmd(timeout time.Duration, command string) ([]string, time.Duration, error) {
	start := time.Now()
	_ = m.conn.SetWriteDeadline(time.Now().Add(timeout))
	if _, err := m.conn.Write([]byte(command + "\n")); err != nil {
		return nil, 0, err
	}
	_ = m.conn.SetReadDeadline(time.Now().Add(timeout))

	var lines []string
	for {
		s, err := m.reader.ReadString('\n')
		if err != nil {
			return nil, 0, err
		}
		s = strings.TrimRight(s, "\r\n")
		if s == "OK" {
			return lines, time.Since(start), nil
		}
		if strings.HasPrefix(s, "ACK") {
			return nil, 0, errors.New(s)
		}
		lines = append(lines, s)
	}
}

// Report Types
type Check struct {
	Name     string `json:"name"`
	OK       bool   `json:"ok"`
	Warning  bool   `json:"warning"`
	Duration int64  `json:"duration"`
	Message  string `json:"message"`
}

type Report struct {
	Host       string  `json:"host"`
	Port       int     `json:"port"`
	TimoutMS   int     `json:"timeout_ms"`
	MPDVersion string  `json:"mpd_version"`
	Checks     []Check `json:"checks"`
	Result     string  `json:"result"`
	ExitCode   int     `json:"exit_code"`
}

const (
	ExitOK         = 0
	ExitNoConnect  = 2
	ExitGreeting   = 3
	ExitCmdFailed  = 4
	ExitDeepFailed = 5
	ExitAuthFailed = 6
	ExitInternal   = 9
)

// Core Logic
func Run(ctx context.Context, cfg Config, deep bool) Report {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	timeout := time.Duration(cfg.TimeoutMS) * time.Millisecond

	rep := Report{
		Host:     cfg.Host,
		Port:     cfg.Port,
		TimoutMS: cfg.TimeoutMS,
		Checks:   []Check{},
		Result:   "FAIL",
		ExitCode: ExitInternal,
	}

	// Connection and greeating
	mpd, ver, d, err := dial(ctx, addr, timeout)
	if err != nil {
		rep.Checks = append(rep.Checks, Check{"tcp_connect", false, false, ms(d), fmt.Sprintf("connect %s failed %v", addr, err)})
		rep.Result = "FAIL(connect)"
		rep.ExitCode = ExitNoConnect
		return rep
	}
	defer mpd.Close()
	rep.Checks = append(rep.Checks, Check{"tcp_connect", true, false, ms(d), "connected"})
	rep.MPDVersion = ver
	rep.Checks = append(rep.Checks, Check{"greeting", true, false, 0, "OK MPD " + ver})

	// Status
	if lines, dur, err := mpd.cmd(timeout, "status"); err != nil {
		code := ExitCmdFailed
		if strings.Contains(err.Error(), "permission denied") {
			code = ExitAuthFailed
		}
		rep.Checks = append(rep.Checks, Check{"status", false, false, ms(dur), err.Error()})
		rep.Result = "FAIL(status)"
		rep.ExitCode = code
		return rep
	} else {
		state := "unknown"
		for _, ln := range lines {
			if strings.HasPrefix(ln, "state: ") {
				state = strings.TrimPrefix(ln, "state: ")
				break
			}
		}
		rep.Checks = append(rep.Checks, Check{"status", true, false, ms(dur), "state=" + state})
	}

	// Stats
	var songs = "unknown"
	if lines, dur, err := mpd.cmd(timeout, "stats"); err != nil {
		rep.Checks = append(rep.Checks, Check{"stats", false, false, ms(dur), err.Error()})
		rep.Result = "FAIL(stats)"
		rep.ExitCode = ExitCmdFailed
		return rep
	} else {
		for _, ln := range lines {
			if strings.HasPrefix(ln, "songs: ") {
				songs = strings.TrimPrefix(ln, "songs: ")
				break
			}
		}
		warn := songs == "0"
		msg := "songs=" + songs
		if warn {
			msg += " (library empty? run mpc update)"
		}
		rep.Checks = append(rep.Checks, Check{"stats", true, warn, ms(dur), msg})
	}

	// Outputs
	if lines, dur, err := mpd.cmd(timeout, "outputs"); err != nil {
		rep.Checks = append(rep.Checks, Check{"outputs", false, false, ms(dur), err.Error()})
		rep.Result = "FAIL(outputs)"
		rep.ExitCode = ExitCmdFailed
		return rep
	} else {
		var total, enabled int
		for _, ln := range lines {
			if strings.HasPrefix(ln, "outputid: ") {
				total++
			}
			if strings.HasPrefix(ln, "outputenabled: ") && strings.HasSuffix(ln, "1") {
				enabled++
			}
		}
		warn := total == 0 || enabled == 0
		msg := fmt.Sprintf("output=%d enabled=%d", total, enabled)
		if warn {
			msg += " (enabled with 'mpc enable <id>')"
		}
		rep.Checks = append(rep.Checks, Check{"output", true, warn, ms(dur), msg})
	}

	// Deep
	if deep {
		if _, dur, err := mpd.cmd(timeout, "idle player database"); err != nil {
			_, _, _ = mpd.cmd(timeout, "noidle")
			rep.Checks = append(rep.Checks, Check{"idle_roundtrip", false, false, ms(dur), "idle failed (try again, or skip --deep)"})
			rep.Result = "FAIL(deep)"
			rep.ExitCode = ExitDeepFailed
			return rep
		}
		_, _, _ = mpd.cmd(timeout, "noidle")
		rep.Checks = append(rep.Checks, Check{"idle_roundtrip", true, false, 0, "idle/noidle OK"})
	}

	rep.Result = "PASS"
	rep.ExitCode = ExitOK
	return rep
}

func ms(d time.Duration) int64 { return d.Milliseconds() }
