package providers_test

import (
	"bufio"
	"errors"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
	"testing"

	"relayhub/internal/providers"
	"relayhub/internal/retry"
)

// --------------------------------------------------------------------------
// Minimal local SMTP test server
//
// Implements just enough of the SMTP protocol to let net/smtp's client
// authenticate and send a message.  No external process or library needed.
//
// Authentication model:
//   net/smtp.PlainAuth sends "AUTH PLAIN <base64(identity\0user\0pass)>" in
//   a single command, so we respond with 235 immediately (one-step PLAIN).
// --------------------------------------------------------------------------

type smtpServer struct {
	listener net.Listener
	addr     string

	mu          sync.Mutex
	receivedTo  []string
	receivedMsg string
	rejectAuth  bool // respond 535 to AUTH
	rejectRcpt  bool // respond 550 to RCPT TO
}

func newSMTPServer(t *testing.T) *smtpServer {
	t.Helper()
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("smtpServer listen: %v", err)
	}
	s := &smtpServer{listener: ln, addr: ln.Addr().String()}
	go s.serve()
	return s
}

func (s *smtpServer) close() { _ = s.listener.Close() }

func (s *smtpServer) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *smtpServer) handleConn(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)

	writeLine := func(line string) {
		_, _ = conn.Write([]byte(line + "\r\n"))
	}
	readLine := func() string {
		line, _ := r.ReadString('\n')
		return strings.TrimRight(line, "\r\n")
	}

	// Greeting
	writeLine("220 localhost ESMTP test")

	for {
		line := readLine()
		if line == "" {
			return
		}
		cmd := strings.ToUpper(line)

		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			// Advertise AUTH PLAIN (one-step: client sends everything in one go)
			writeLine("250-localhost")
			writeLine("250 AUTH PLAIN")

		case strings.HasPrefix(cmd, "AUTH PLAIN"):
			// net/smtp sends "AUTH PLAIN <b64token>" in a single line.
			// If no token is on the line, we'd need to issue a 334 challenge,
			// but net/smtp always includes the token, so we just accept/reject.
			s.mu.Lock()
			reject := s.rejectAuth
			s.mu.Unlock()
			if reject {
				writeLine("535 5.7.8 Authentication credentials invalid")
				return
			}
			writeLine("235 2.7.0 Authentication successful")

		case strings.HasPrefix(cmd, "MAIL FROM"):
			writeLine("250 OK")

		case strings.HasPrefix(cmd, "RCPT TO"):
			s.mu.Lock()
			reject := s.rejectRcpt
			s.mu.Unlock()
			if reject {
				writeLine("550 5.1.1 No such user")
				return
			}
			start := strings.Index(line, "<")
			end := strings.Index(line, ">")
			if start >= 0 && end > start {
				s.mu.Lock()
				s.receivedTo = append(s.receivedTo, line[start+1:end])
				s.mu.Unlock()
			}
			writeLine("250 OK")

		case cmd == "DATA":
			writeLine("354 End data with <CR><LF>.<CR><LF>")
			var sb strings.Builder
			for {
				chunk := readLine()
				if chunk == "." {
					break
				}
				sb.WriteString(chunk + "\n")
			}
			s.mu.Lock()
			s.receivedMsg = sb.String()
			s.mu.Unlock()
			writeLine("250 OK queued")

		case cmd == "QUIT":
			writeLine("221 Bye")
			return

		default:
			writeLine("502 Command not implemented")
		}
	}
}

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func TestSMTPProvider_Name(t *testing.T) {
	p := providers.NewSMTPProvider("localhost", "587", "", "", "from@example.com")
	if p.Name() != "smtp" {
		t.Fatalf("expected Name() == %q, got %q", "smtp", p.Name())
	}
}

func TestSMTPProvider_Send_Success(t *testing.T) {
	srv := newSMTPServer(t)
	defer srv.close()

	host, port, _ := net.SplitHostPort(srv.addr)
	p := providers.NewSMTPProvider(host, port, "user", "pass", "relay@example.com")

	if err := p.Send("alice@example.com", "Hello via SMTP!"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	srv.mu.Lock()
	defer srv.mu.Unlock()
	if len(srv.receivedTo) == 0 || srv.receivedTo[0] != "alice@example.com" {
		t.Errorf("expected RCPT TO alice@example.com, got %v", srv.receivedTo)
	}
	if !strings.Contains(srv.receivedMsg, "Hello via SMTP!") {
		t.Errorf("expected body to contain 'Hello via SMTP!', got:\n%s", srv.receivedMsg)
	}
}

func TestSMTPProvider_InvalidCredentials(t *testing.T) {
	srv := newSMTPServer(t)
	defer srv.close()

	srv.mu.Lock()
	srv.rejectAuth = true
	srv.mu.Unlock()

	host, port, _ := net.SplitHostPort(srv.addr)
	p := providers.NewSMTPProvider(host, port, "baduser", "badpass", "relay@example.com")

	err := p.Send("alice@example.com", "Should fail")
	if err == nil {
		t.Fatal("expected auth error, got nil")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("expected 'authentication failed' in error, got: %v", err)
	}
}

func TestSMTPProvider_InvalidRecipient(t *testing.T) {
	srv := newSMTPServer(t)
	defer srv.close()

	srv.mu.Lock()
	srv.rejectRcpt = true
	srv.mu.Unlock()

	host, port, _ := net.SplitHostPort(srv.addr)
	p := providers.NewSMTPProvider(host, port, "user", "pass", "relay@example.com")

	err := p.Send("nobody@invalid.example", "Should fail")
	if err == nil {
		t.Fatal("expected RCPT error, got nil")
	}
	if !strings.Contains(err.Error(), "RCPT TO rejected") {
		t.Errorf("expected 'RCPT TO rejected' in error, got: %v", err)
	}
}

func TestSMTPProvider_ConnectionRefused(t *testing.T) {
	p := providers.NewSMTPProvider("127.0.0.1", "19999", "user", "pass", "relay@example.com")
	err := p.Send("alice@example.com", "Hello")
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "connection refused") &&
		!strings.Contains(lower, "timeout") &&
		!strings.Contains(lower, "connect") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSMTPProvider_MissingHost(t *testing.T) {
	p := providers.NewSMTPProvider("", "587", "user", "pass", "relay@example.com")
	err := p.Send("alice@example.com", "Hello")
	if err == nil || !strings.Contains(err.Error(), "SMTP_HOST") {
		t.Fatalf("expected SMTP_HOST error, got: %v", err)
	}
}

func TestSMTPProvider_MissingFrom(t *testing.T) {
	p := providers.NewSMTPProvider("localhost", "587", "user", "pass", "")
	err := p.Send("alice@example.com", "Hello")
	if err == nil || !strings.Contains(err.Error(), "SMTP_FROM") {
		t.Fatalf("expected SMTP_FROM error, got: %v", err)
	}
}

func TestSMTPProvider_MissingRecipient(t *testing.T) {
	p := providers.NewSMTPProvider("localhost", "587", "user", "pass", "relay@example.com")
	err := p.Send("", "Hello")
	if err == nil || !strings.Contains(err.Error(), "recipient") {
		t.Fatalf("expected recipient error, got: %v", err)
	}
}

func TestSMTPProvider_InvalidRecipientFormat(t *testing.T) {
	p := providers.NewSMTPProvider("localhost", "587", "user", "pass", "relay@example.com")
	err := p.Send("notanemail", "Hello")
	if err == nil || !strings.Contains(err.Error(), "valid email") {
		t.Fatalf("expected 'valid email' error, got: %v", err)
	}
}

func TestSMTPProvider_RoutingThroughRetry_ImmediateSuccess(t *testing.T) {
	srv := newSMTPServer(t)
	defer srv.close()

	host, port, _ := net.SplitHostPort(srv.addr)
	p := providers.NewSMTPProvider(host, port, "user", "pass", "relay@example.com")
	logger := newLogger()

	attempts, err := retry.WithRetry(func() error {
		return p.Send("bob@example.com", "retry test")
	}, 3, logger)

	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestSMTPProvider_RoutingThroughRetry_EventualSuccess(t *testing.T) {
	logger := newLogger()
	calls := 0

	srv := newSMTPServer(t)
	defer srv.close()
	host, port, _ := net.SplitHostPort(srv.addr)
	p := providers.NewSMTPProvider(host, port, "user", "pass", "relay@example.com")

	attempts, err := retry.WithRetry(func() error {
		calls++
		if calls < 3 {
			return errors.New("smtp: transient error (simulated)")
		}
		return p.Send("bob@example.com", "retry test")
	}, 3, logger)

	if err != nil {
		t.Fatalf("expected success on attempt 3, got: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}
