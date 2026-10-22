// Package surel mengirim email transaksional (atur ulang password) lewat SMTP biasa.
// Konfigurasi dari env: SMTP_HOST, SMTP_PORT (bawaan 587), SMTP_USER, SMTP_PASSWORD, SMTP_FROM.
// net/smtp memakai STARTTLS otomatis bila server menawarkannya; autentikasi PLAIN hanya dikirim
// lewat koneksi terenkripsi atau localhost.
package surel

import (
	"cmp"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"os"
	"strings"
)

type Config struct{ Host, Port, User, Password, From string }

func ConfigFromEnv() Config {
	return Config{
		Host: strings.TrimSpace(os.Getenv("SMTP_HOST")), Port: cmp.Or(strings.TrimSpace(os.Getenv("SMTP_PORT")), "587"),
		User: os.Getenv("SMTP_USER"), Password: os.Getenv("SMTP_PASSWORD"), From: strings.TrimSpace(os.Getenv("SMTP_FROM")),
	}
}

// Aktif: tanpa host & pengirim, email tidak dikirim (pemanggil mencatat ke log saja).
func (c Config) Aktif() bool { return c.Host != "" && c.From != "" }

// Kirim satu email teks biasa.
func (c Config) Kirim(ke, subjek, isi string) error {
	if !c.Aktif() {
		return errors.New("SMTP belum diatur")
	}
	if strings.ContainsAny(ke+subjek, "\r\n") {
		return errors.New("alamat atau subjek tidak valid")
	}
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		c.From, ke, mime.QEncoding.Encode("utf-8", subjek), strings.ReplaceAll(isi, "\n", "\r\n"))
	var auth smtp.Auth
	if c.User != "" {
		auth = smtp.PlainAuth("", c.User, c.Password, c.Host)
	}
	return smtp.SendMail(net.JoinHostPort(c.Host, c.Port), auth, c.From, []string{ke}, []byte(msg))
}
