package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/mailer"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
	"net/mail"
	"net/smtp"
	"net/textproto"
)

// SSLType is the kind of TLS used on the SMTP connection.
type SSLType uint8

const (
	// SSLNone opens a plain unencrypted connection.
	SSLNone SSLType = iota

	// SSLTLS opens an SSL (TLS) connection directly without STARTTLS.
	SSLTLS

	// SSLSTARTTLS opens a plain connection then upgrades it via STARTTLS.
	SSLSTARTTLS
)

const (
	defaultSweepInterval = 2 * time.Second
	defaultWaitTimeout   = 2 * time.Second
)

var (
	// ErrSMTPClosed is returned when the pool is closed.
	ErrSMTPClosed = errors.New("smtp pool closed")
)

// SMTP is a pool of reusable SMTP connections. It creates connections lazily,
// reuses them across messages, and sweeps idle connections after IdleTimeout.
// Design is modeled after github.com/knadh/smtppool.
type SMTP struct {
	from     string
	fromName string

	conns        chan *smtpConn
	createdConns atomic.Int32
	stopBorrow   chan bool
	closed       atomic.Bool

	opt smtpOpt
}

type smtpOpt struct {
	addr        string
	host        string
	helloHost   string
	maxConns    int
	maxRetries  int
	idleTimeout time.Duration
	waitTimeout time.Duration
	ssl         SSLType
	tlsConfig   *tls.Config
	user        string
	password    string
}

// smtpConn is an SMTP client together with the time of its last activity.
type smtpConn struct {
	conn         *smtp.Client
	lastActivity time.Time
}

// NewSMTP builds a pooled SMTP sender from cfg. from is the envelope sender.
func NewSMTP(from, fromName string, cfg config.SMTPConfig) (*SMTP, error) {
	if cfg.Host == "" {
		return nil, errors.New("SMTP_HOST is required")
	}
	if cfg.PoolSize < 1 {
		cfg.PoolSize = 1
	}

	ssl, err := parseSSLType(cfg.SSL)
	if err != nil {
		return nil, err
	}

	opt := smtpOpt{
		addr:        fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		host:        cfg.Host,
		maxConns:    cfg.PoolSize,
		maxRetries:  2,
		idleTimeout: 30 * time.Second,
		waitTimeout: defaultWaitTimeout,
		ssl:         ssl,
		user:        cfg.User,
		password:    cfg.Password,
	}

	p := &SMTP{
		from:       from,
		fromName:   fromName,
		conns:      make(chan *smtpConn, opt.maxConns),
		stopBorrow: make(chan bool),
		opt:        opt,
	}

	// Sweep idle connections in the background.
	go p.sweepConns(defaultSweepInterval)
	return p, nil
}

// parseSSLType maps the config string to an SSLType.
func parseSSLType(s string) (SSLType, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "starttls":
		return SSLSTARTTLS, nil
	case "none":
		return SSLNone, nil
	case "tls", "ssl":
		return SSLTLS, nil
	default:
		return SSLNone, fmt.Errorf("invalid MAIL_SMTP_SSL %q (want none, tls, starttls)", s)
	}
}

// Send implements mailer.MailSender. On failure the message is retried on a
// fresh connection when the error is retriable.
func (s *SMTP) Send(ctx context.Context, msg mailer.Message) error {
	data, err := buildMIME(s.from, s.fromName, msg)
	if err != nil {
		return err
	}

	var lastErr error
	for i := 0; i < s.opt.maxRetries; i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		c, err := s.borrowConn(ctx)
		if err != nil {
			lastErr = err
			if canRetry(err) {
				continue
			}
			return err
		}

		retry, err := s.sendOnConn(c, s.from, msg.To, data)
		if err == nil {
			_ = s.returnConn(c, nil)
			return nil
		}
		lastErr = err
		_ = s.returnConn(c, err)
		if !retry {
			return err
		}
	}
	return lastErr
}

// Close closes the pool. The background sweeper drains and quits on its next
// pass; borrows in flight return ErrSMTPClosed.
func (s *SMTP) Close() {
	if s.closed.CompareAndSwap(false, true) {
		close(s.stopBorrow)
	}
}

// newConn dials, optionally upgrades to TLS/STARTTLS, and authenticates a
// fresh SMTP connection.
func (s *SMTP) newConn() (*smtpConn, error) {
	var (
		netCon net.Conn
		err    error
	)

	switch s.opt.ssl {
	case SSLTLS:
		// Direct TLS connection (e.g. port 465).
		c, derr := tls.DialWithDialer(&net.Dialer{Timeout: s.opt.waitTimeout}, "tcp", s.opt.addr, s.opt.tlsConfig)
		if derr != nil {
			return nil, derr
		}
		netCon = c

	default: // SSLNone, SSLSTARTTLS
		c, derr := net.DialTimeout("tcp", s.opt.addr, s.opt.waitTimeout)
		if derr != nil {
			return nil, derr
		}
		netCon = c
	}

	cl, err := smtp.NewClient(netCon, s.opt.host)
	if err != nil {
		netCon.Close()
		return nil, err
	}

	// Close the client on any error from here onwards.
	defer func() {
		if err != nil {
			cl.Close()
		}
	}()

	if s.opt.helloHost != "" {
		_ = cl.Hello(s.opt.helloHost)
	}

	if s.opt.ssl == SSLSTARTTLS {
		if ok, _ := cl.Extension("STARTTLS"); !ok {
			return nil, errors.New("smtp: STARTTLS extension not supported by server")
		}
		if err = cl.StartTLS(s.opt.tlsConfig); err != nil {
			return nil, err
		}
	}

	if s.opt.user != "" || s.opt.password != "" {
		if ok, _ := cl.Extension("AUTH"); !ok {
			return nil, errors.New("smtp: AUTH extension not supported by server")
		}
		auth := smtp.PlainAuth("", s.opt.user, s.opt.password, s.opt.host)
		if err = cl.Auth(auth); err != nil {
			return nil, err
		}
	}

	return &smtpConn{conn: cl, lastActivity: time.Now()}, nil
}

// borrowConn returns a connection from the pool, creating a new one when there
// is room.
func (s *SMTP) borrowConn(ctx context.Context) (*smtpConn, error) {
	switch {
	case s.closed.Load():
		return nil, ErrSMTPClosed
	case int(s.createdConns.Load()) < s.opt.maxConns && len(s.conns) == 0:
		s.createdConns.Add(1)
		c, err := s.newConn()
		if err != nil {
			s.createdConns.Add(-1)
			return nil, err
		}
		return c, nil
	}

	select {
	case c := <-s.conns:
		return c, nil
	case <-s.stopBorrow:
		return nil, ErrSMTPClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(s.opt.waitTimeout):
		return nil, errors.New("smtp: timed out waiting for a free connection")
	}
}

// returnConn puts the connection back if it is healthy, otherwise closes it.
func (s *SMTP) returnConn(c *smtpConn, lastErr error) (err error) {
	defer func() {
		if err != nil {
			s.createdConns.Add(-1)
			c.conn.Close()
		}
	}()

	if lastErr != nil {
		if err, ok := lastErr.(*textproto.Error); !ok {
			return lastErr
		} else if err.Code == 421 {
			return lastErr
		}
	}

	// RSET before reuse so servers don't complain about command ordering.
	if err := c.conn.Reset(); err != nil {
		return err
	}

	select {
	case s.conns <- c:
		return nil
	case <-s.stopBorrow:
		return ErrSMTPClosed
	case <-time.After(s.opt.waitTimeout):
		return errors.New("smtp: timed out returning a connection to the pool")
	}
}

// sweepConns closes connections idle longer than idleTimeout. It doubles as the
// Close cleanup when called with interval 0. Runs as a goroutine after NewSMTP.
func (s *SMTP) sweepConns(interval time.Duration) {
	active := make([]*smtpConn, 0, cap(s.conns))
	for {
		if interval > 0 {
			time.Sleep(interval)
		}
		active = active[:0]

		var (
			num          = len(s.conns)
			createdConns = s.createdConns.Load()
			closed       = s.closed.Load()
		)
		if closed && createdConns == 0 {
			return
		}

		for i := 0; i < num; i++ {
			var c *smtpConn
			select {
			case c = <-s.conns:
			default:
				continue
			}

			if closed || time.Since(c.lastActivity) > s.opt.idleTimeout {
				s.createdConns.Add(-1)
				if closed {
					_ = c.conn.Quit()
				} else {
					_ = c.conn.Close()
				}
				continue
			}
			active = append(active, c)
		}

		for _, c := range active {
			select {
			case s.conns <- c:
			default:
				_ = c.conn.Close()
				s.createdConns.Add(-1)
			}
		}

		if interval <= 0 {
			return
		}
	}
}

// sendOnConn sends one message on an existing connection. The bool indicates
// whether the message can be retried.
func (s *SMTP) sendOnConn(c *smtpConn, from string, to []string, data []byte) (bool, error) {
	c.lastActivity = time.Now()

	// Normalize recipient addresses.
	recipients, err := normalizeRecipients(to)
	if err != nil {
		return true, err
	}

	if err := c.conn.Mail(from); err != nil {
		return canRetry(err), err
	}
	for _, r := range recipients {
		if err := c.conn.Rcpt(r); err != nil {
			return canRetry(err), err
		}
	}

	w, err := c.conn.Data()
	if err != nil {
		return canRetry(err), err
	}
	if _, err = w.Write(data); err != nil {
		w.Close()
		return canRetry(err), err
	}
	if err := w.Close(); err != nil {
		return false, err
	}
	return false, nil
}

// normalizeRecipients parses bare addresses out of "Name <a@b.c>" strings.
func normalizeRecipients(to []string) ([]string, error) {
	out := make([]string, 0, len(to))
	for _, e := range to {
		addr, err := mail.ParseAddress(e)
		if err != nil {
			return nil, err
		}
		out = append(out, addr.Address)
	}
	return out, nil
}

// canRetry reports whether an error is network-related and therefore the
// message may be retried on a fresh connection.
func canRetry(err error) bool {
	if err == io.EOF {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var opErr *net.OpError
	return errors.As(err, &opErr)
}
