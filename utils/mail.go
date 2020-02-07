//
// mail.go
// Copyright (C) 2018 Odin <Odin@Odin-Pro.local>
//
// Distributed under terms of the MIT license.
//

package utils

import (
	"gopkg.in/gomail.v2"
)

func SendMail(smtp string, port int, account, password, from, to, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(smtp, port, account, password)

	return d.DialAndSend(m)
}

func SendMails(smtp string, port int, account, password, from, subject, body string, addrs ...[]string) error {
	ccs := []string{}
	tos := addrs[0]
	if len(addrs) > 1 {
		ccs = addrs[1]
	}
	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", tos...)
	if len(ccs) > 0 {
		m.SetHeader("Cc", ccs...)
	}
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(smtp, port, account, password)

	return d.DialAndSend(m)
}
