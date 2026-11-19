package mailer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

var ErrUnavailable = errors.New("SMTP delivery unavailable")
var ErrInvalidRecipient = errors.New("invalid email recipient")
var ErrInvalidMessage = errors.New("invalid email message")

type Message struct {
	To        string
	ID        string
	Subject   string
	Body      string
	CreatedAt time.Time
}

type SMTP struct {
	address, host string
	from          *mail.Address
}

// This adapter targets the selected Mailpit test inbox: plain SMTP, no auth.
func NewSMTP(address, from string) (*SMTP, error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil || host == "" {
		return nil, ErrUnavailable
	}
	sender, err := mail.ParseAddress(from)
	if err != nil || strings.ContainsAny(from, "\r\n") {
		return nil, ErrInvalidMessage
	}
	return &SMTP{address: address, host: host, from: sender}, nil
}

func (s *SMTP) Send(ctx context.Context, message Message) error {
	recipient, err := mail.ParseAddress(message.To)
	if err != nil || recipient.Address != message.To || strings.ContainsAny(message.To, "\r\n") {
		return ErrInvalidRecipient
	}
	if message.ID == "" || !strings.HasPrefix(message.ID, "<") || !strings.HasSuffix(message.ID, ">") || strings.ContainsAny(message.ID+message.Subject, "\r\n") {
		return ErrInvalidMessage
	}
	var body bytes.Buffer
	encoded := quotedprintable.NewWriter(&body)
	if _, err := encoded.Write([]byte(message.Body)); err != nil {
		return ErrInvalidMessage
	}
	if err := encoded.Close(); err != nil {
		return ErrInvalidMessage
	}
	var wire bytes.Buffer
	fmt.Fprintf(&wire, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMessage-ID: %s\r\nDate: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n", s.from.String(), recipient.String(), message.Subject, message.ID, message.CreatedAt.Format(time.RFC1123Z))
	wire.Write(body.Bytes())
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	connection, err := (&net.Dialer{}).DialContext(ctx, "tcp", s.address)
	if err != nil {
		return ErrUnavailable
	}
	defer func() { _ = connection.Close() }()
	deadline, _ := ctx.Deadline()
	if err := connection.SetDeadline(deadline); err != nil {
		return ErrUnavailable
	}
	stop := context.AfterFunc(ctx, func() { _ = connection.Close() })
	defer stop()
	client, err := smtp.NewClient(connection, s.host)
	if err != nil {
		return ErrUnavailable
	}
	defer func() { _ = client.Close() }()
	if err := client.Mail(s.from.Address); err != nil {
		return ErrUnavailable
	}
	if err := client.Rcpt(recipient.Address); err != nil {
		return ErrUnavailable
	}
	data, err := client.Data()
	if err != nil {
		return ErrUnavailable
	}
	if _, err := data.Write(wire.Bytes()); err != nil {
		return ErrUnavailable
	}
	// Close reads the final DATA response: only an accepted message is successful.
	if err := data.Close(); err != nil {
		return ErrUnavailable
	}
	if err := client.Quit(); err != nil {
		// DATA was accepted. A cleanup failure must not trigger duplicate delivery.
		_ = client.Close()
	}
	return nil
}
