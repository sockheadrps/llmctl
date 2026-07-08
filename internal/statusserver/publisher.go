package statusserver

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Publisher pushes local client-mode snapshots to a remote status server.
type Publisher struct {
	addr string
	id   string
	name string

	mu     sync.Mutex
	cancel context.CancelFunc
	latest Status
	ch     chan Status
}

// NewPublisher creates an idle publisher. Call Start before Update.
func NewPublisher(id, name string) *Publisher {
	return &Publisher{id: id, name: name}
}

// Start connects the publisher loop to addr, a host:port remote status server.
func (p *Publisher) Start(addr string) {
	p.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	p.mu.Lock()
	p.addr = addr
	p.cancel = cancel
	p.ch = make(chan Status, 1)
	if p.latest.Version != "" || len(p.latest.Running) > 0 || p.latest.GPU != nil {
		p.ch <- p.latest
	}
	ch := p.ch
	p.mu.Unlock()

	go p.run(ctx, addr, ch)
}

// Stop closes the background publisher loop.
func (p *Publisher) Stop() {
	p.mu.Lock()
	cancel := p.cancel
	p.cancel = nil
	p.ch = nil
	p.addr = ""
	p.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Update publishes the latest snapshot. It never blocks the caller on network
// I/O; when the loop is busy, the newest snapshot replaces the queued one.
func (p *Publisher) Update(st Status) {
	p.mu.Lock()
	p.latest = st
	ch := p.ch
	p.mu.Unlock()
	if ch == nil {
		return
	}
	select {
	case ch <- st:
	default:
		select {
		case <-ch:
		default:
		}
		select {
		case ch <- st:
		default:
		}
	}
}

func (p *Publisher) run(ctx context.Context, addr string, ch <-chan Status) {
	var conn *websocket.Conn
	defer func() {
		if conn != nil {
			_ = conn.Close()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case st := <-ch:
			for {
				if conn == nil {
					c, err := dialClientStatusWS(ctx, addr)
					if err != nil {
						if !sleepOrDone(ctx, 2*time.Second) {
							return
						}
						continue
					}
					conn = c
				}
				msg := clientUpdate{ID: p.id, Name: p.name, Status: st}
				_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
				if err := conn.WriteJSON(msg); err != nil {
					_ = conn.Close()
					conn = nil
					if !sleepOrDone(ctx, 500*time.Millisecond) {
						return
					}
					continue
				}
				break
			}
		}
	}
}

func dialClientStatusWS(ctx context.Context, addr string) (*websocket.Conn, error) {
	u := url.URL{Scheme: "ws", Host: strings.TrimSpace(addr), Path: "/ws/client-status"}
	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	conn, _, err := dialer.DialContext(ctx, u.String(), nil)
	return conn, err
}

func sleepOrDone(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
