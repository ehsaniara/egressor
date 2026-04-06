// Package proxy implements an HTTPS intercepting proxy with TLS MITM
// for monitoring and controlling outbound traffic.
package proxy

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/ehsaniara/egressor/internal/audit"
)

type Server struct {
	listenAddr  string
	logger      audit.SessionSink
	interceptor *Interceptor

	mu      sync.Mutex
	cancel  context.CancelFunc
	running bool
}

func NewServer(listenAddr string, logger audit.SessionSink, interceptor *Interceptor) *Server {
	return &Server{
		listenAddr:  listenAddr,
		logger:      logger,
		interceptor: interceptor,
	}
}

// Start begins listening in a background goroutine.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("server already running")
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.running = true

	errCh := make(chan error, 1)
	go func() {
		err := s.ListenAndServe(ctx)
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		if err != nil {
			errCh <- err
		}
		close(errCh)
	}()

	// Give the listener a moment to bind or fail
	select {
	case err := <-errCh:
		s.mu.Lock()
		s.running = false
		s.cancel = nil
		s.mu.Unlock()
		return err
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}

// Stop shuts down the proxy server.
func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

// IsRunning returns whether the proxy is currently listening.
func (s *Server) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// Address returns the configured listen address.
func (s *Server) Address() string {
	return s.listenAddr
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				slog.Error("accept error", "err", err)
				continue
			}
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Read the request line: CONNECT host:port HTTP/1.1
	requestLine, err := reader.ReadString('\n')
	if err != nil {
		slog.Debug("failed to read request line", "err", err)
		return
	}
	requestLine = strings.TrimSpace(requestLine)

	parts := strings.Fields(requestLine)
	if len(parts) < 3 {
		writeResponse(conn, "400 Bad Request")
		return
	}

	method, target := parts[0], parts[1]

	// Drain headers
	for {
		line, err := reader.ReadString('\n')
		if err != nil || strings.TrimSpace(line) == "" {
			break
		}
	}

	if strings.ToUpper(method) != "CONNECT" {
		writeResponse(conn, "405 Method Not Allowed")
		return
	}

	host, port, err := parseTarget(target)
	if err != nil {
		writeResponse(conn, "400 Bad Request")
		return
	}

	sess := audit.NewSession(conn.RemoteAddr().String(), host, port)

	upstream, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 5*time.Second)
	if err != nil {
		sess.DialStatus = "failed"
		sess.Error = err.Error()
		sess.Finish()
		s.logger.Log(sess)
		writeResponse(conn, "502 Bad Gateway")
		slog.Warn("dial failed", "host", host, "port", port, "err", err)
		return
	}
	defer upstream.Close()

	sess.DialStatus = "success"
	writeResponse(conn, "200 Connection Established")

	slog.Info("intercepting", "host", host, "port", port, "session", sess.ID)
	if err := s.interceptor.Intercept(conn, upstream, host, sess); err != nil {
		slog.Warn("intercept error", "session", sess.ID, "err", err)
		sess.Error = err.Error()
	}
	sess.Finish()
	s.logger.Log(sess)
	slog.Info("session closed", "session", sess.ID, "exchanges", len(sess.Exchanges), "duration_ms", sess.DurationMs)
}

func writeResponse(conn net.Conn, status string) {
	fmt.Fprintf(conn, "HTTP/1.1 %s\r\n\r\n", status)
}

func parseTarget(target string) (host, port string, err error) {
	host, port, err = net.SplitHostPort(target)
	if err != nil {
		// No port specified, default to 443
		host = target
		port = "443"
		err = nil
	}
	if host == "" {
		return "", "", fmt.Errorf("empty host")
	}
	return host, port, nil
}
