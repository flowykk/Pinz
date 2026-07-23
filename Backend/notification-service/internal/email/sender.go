package email

import (
	"fmt"
	"net/smtp"
	"os"
)

type Sender struct {
	host string
	port string
	username string
	password string
	from string
}

// NewSenderFromEnv возвращает nil, если SMTP_HOST не задан.
func NewSenderFromEnv() *Sender {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		return nil
	}
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	return &Sender{
		host: host,
		port: port,
		username: os.Getenv("SMTP_USERNAME"),
		password: os.Getenv("SMTP_PASSWORD"),
		from: os.Getenv("SMTP_FROM"),
	}
}

func NewSender(host, port, username, password, from string) *Sender {
	return &Sender{host: host, port: port, username: username, password: password, from: from}
}

func (s *Sender) SendVerificationCode(to, code string) error {
	if s == nil {
		return nil
	}
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: Pinz: your verification code\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nYour verification code is: %s\r\n", s.from, to, code)

	return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg))
}
