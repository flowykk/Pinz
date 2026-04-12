package email

import (
	"fmt"
	"net/smtp"
)

type Sender struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewSender(host, port, username, password, from string) *Sender {
	return &Sender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (s *Sender) SendVerificationCode(to, code string) error {
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: Pinz: your verification code\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nYour verification code is: %s\r\n", s.from, to, code)

	return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg))
}
