// Package surel mengirim email transaksional (atur ulang password) lewat SMTP.
// Konfigurasi dari env: SMTP_HOST, SMTP_PORT (bawaan 587), SMTP_USER, SMTP_PASSWORD, SMTP_FROM.
// Koneksi wajib terenkripsi: port 465 memakai TLS langsung, port lain wajib STARTTLS (kecuali
// relay di localhost). Ada batas waktu supaya server SMTP yang macet tidak menahan proses.
package surel

import (
	"cmp"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"
)

const batasWaktu = 20 * time.Second

type Config struct{ Host, Port, User, Password, From string }

func ConfigFromEnv() Config {
	return Config{
		Host: strings.TrimSpace(os.Getenv("SMTP_HOST")), Port: cmp.Or(strings.TrimSpace(os.Getenv("SMTP_PORT")), "587"),
		User: os.Getenv("SMTP_USER"), Password: os.Getenv("SMTP_PASSWORD"), From: strings.TrimSpace(os.Getenv("SMTP_FROM")),
	}
}

// Aktif: tanpa host & pengirim, email tidak dikirim.
func (c Config) Aktif() bool { return c.Host != "" && c.From != "" }

// Kirim satu email teks biasa.
func (c Config) Kirim(ke, subjek, isi string) error {
	if !c.Aktif() {
		return errors.New("SMTP belum diatur")
	}
	if strings.ContainsAny(ke+subjek+c.From, "\r\n") {
		return errors.New("alamat atau subjek tidak valid")
	}
	domain := c.From[strings.LastIndex(c.From, "@")+1:]
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMessage-ID: <%d@%s>\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		c.From, ke, mime.QEncoding.Encode("utf-8", subjek), time.Now().Format(time.RFC1123Z), time.Now().UnixNano(), strings.Trim(domain, "<> "),
		strings.ReplaceAll(isi, "\n", "\r\n"))
	alamat := net.JoinHostPort(c.Host, c.Port)
	d := net.Dialer{Timeout: batasWaktu}
	var conn net.Conn
	var err error
	if c.Port == "465" {
		conn, err = tls.DialWithDialer(&d, "tcp", alamat, &tls.Config{ServerName: c.Host})
	} else {
		conn, err = d.Dial("tcp", alamat)
	}
	if err != nil {
		return err
	}
	_ = conn.SetDeadline(time.Now().Add(batasWaktu))
	cl, err := smtp.NewClient(conn, c.Host)
	if err != nil {
		conn.Close()
		return err
	}
	defer cl.Close()
	if c.Port != "465" {
		if ok, _ := cl.Extension("STARTTLS"); ok {
			if err := cl.StartTLS(&tls.Config{ServerName: c.Host}); err != nil {
				return err
			}
		} else if c.Host != "localhost" && c.Host != "127.0.0.1" {
			return errors.New("server SMTP tidak menawarkan STARTTLS; email tidak dikirim tanpa enkripsi")
		}
	}
	if c.User != "" {
		if err := cl.Auth(smtp.PlainAuth("", c.User, c.Password, c.Host)); err != nil {
			return err
		}
	}
	if err := cl.Mail(c.From); err != nil {
		return err
	}
	if err := cl.Rcpt(ke); err != nil {
		return err
	}
	w, err := cl.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return cl.Quit()
}
