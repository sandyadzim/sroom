package mailer

import (
	"bufio"
	"context"
	"errors"
	"io"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/textproto"
	"strings"
	"testing"
	"time"
)

func smtpTestServer(t *testing.T, behavior string) (string, <-chan string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	received := make(chan string, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		connection, err := listener.Accept()
		if err != nil {
			return
		}
		defer connection.Close()
		if behavior == "hang" {
			_, _ = io.Copy(io.Discard, connection)
			return
		}
		reader := textproto.NewReader(bufio.NewReader(connection))
		_, _ = io.WriteString(connection, "220 local test SMTP\r\n")
		for {
			line, err := reader.ReadLine()
			if err != nil {
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"), strings.HasPrefix(line, "MAIL FROM"):
				_, _ = io.WriteString(connection, "250 OK\r\n")
			case strings.HasPrefix(line, "RCPT TO"):
				if behavior == "reject" {
					_, _ = io.WriteString(connection, "550 rejected synthetic recipient\r\n")
				} else {
					_, _ = io.WriteString(connection, "250 OK\r\n")
				}
			case line == "DATA":
				_, _ = io.WriteString(connection, "354 Send data\r\n")
				body, err := reader.ReadDotBytes()
				if err != nil {
					return
				}
				received <- string(body)
				_, _ = io.WriteString(connection, "250 Accepted\r\n")
			case line == "QUIT":
				if behavior != "quit fails" {
					_, _ = io.WriteString(connection, "221 Bye\r\n")
				}
				return
			default:
				_, _ = io.WriteString(connection, "500 unexpected test command\r\n")
			}
		}
	}()
	t.Cleanup(func() {
		_ = listener.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("test SMTP did not stop")
		}
	})
	return listener.Addr().String(), received
}

func TestSMTPAcceptanceAndMIME(t *testing.T) {
	for _, behavior := range []string{"accepted", "quit fails"} {
		t.Run(behavior, func(t *testing.T) {
			address, received := smtpTestServer(t, behavior)
			sender, err := NewSMTP(address, "SROOM <noreply@sroom.test>")
			if err != nil {
				t.Fatal(err)
			}
			message := Message{To: "customer@example.test", ID: "<synthetic@sroom.test>", Subject: "SROOM invoice", Body: "Synthetic car — café\nSign in required.\n", CreatedAt: time.Now().UTC()}
			if err := sender.Send(t.Context(), message); err != nil {
				t.Fatal(err)
			}
			parsed, err := mail.ReadMessage(strings.NewReader(<-received))
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(quotedprintable.NewReader(parsed.Body))
			if err != nil {
				t.Fatal(err)
			}
			if parsed.Header.Get("Message-ID") != message.ID || parsed.Header.Get("Content-Type") != "text/plain; charset=UTF-8" || !strings.Contains(string(body), "café") || strings.Contains(string(body), "Content-Disposition: attachment") {
				t.Fatal("message encoding or headers incorrect")
			}
		})
	}
}

func TestSMTPRejectAndCancellation(t *testing.T) {
	for _, behavior := range []string{"reject", "hang"} {
		t.Run(behavior, func(t *testing.T) {
			address, _ := smtpTestServer(t, behavior)
			sender, err := NewSMTP(address, "noreply@sroom.test")
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
			defer cancel()
			started := time.Now()
			err = sender.Send(ctx, Message{To: "customer@example.test", ID: "<synthetic@sroom.test>", Subject: "Synthetic", Body: "Synthetic", CreatedAt: time.Now()})
			if !errors.Is(err, ErrUnavailable) || time.Since(started) > time.Second || strings.Contains(err.Error(), "customer@example.test") {
				t.Fatal("SMTP failure was not bounded and sanitized")
			}
		})
	}
}

func TestSMTPValidation(t *testing.T) {
	sender, err := NewSMTP("127.0.0.1:1", "noreply@sroom.test")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		to, subject, id string
		want            error
	}{
		{"bad", "Subject", "<test@sroom.test>", ErrInvalidRecipient},
		{"customer@example.test\r\nBcc: other@example.test", "Subject", "<test@sroom.test>", ErrInvalidRecipient},
		{"customer@example.test", "Subject\r\nBcc: other@example.test", "<test@sroom.test>", ErrInvalidMessage},
	} {
		err := sender.Send(t.Context(), Message{To: test.to, Subject: test.subject, ID: test.id})
		if !errors.Is(err, test.want) {
			t.Fatalf("error=%v want=%v", err, test.want)
		}
	}
}
