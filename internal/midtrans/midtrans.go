// Package midtrans: klien minimal Snap + Status API + verifikasi notifikasi, tanpa dependensi.
// Dok: https://docs.midtrans.com/reference/snap-api, https://docs.midtrans.com/docs/https-notification-webhooks
package midtrans

import (
	"bytes"
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	ServerKey  string
	Production bool
	// SnapURL & APIURL bisa ditimpa untuk test; kosong = sesuai Production.
	SnapURL, APIURL string
	HTTP            *http.Client
}

func (c *Client) snapURL() string {
	switch {
	case c.SnapURL != "":
		return c.SnapURL
	case c.Production:
		return "https://app.midtrans.com"
	}
	return "https://app.sandbox.midtrans.com"
}

func (c *Client) apiURL() string {
	switch {
	case c.APIURL != "":
		return c.APIURL
	case c.Production:
		return "https://api.midtrans.com"
	}
	return "https://api.sandbox.midtrans.com"
}

// SnapJS: URL snap.js untuk frontend (sandbox/produksi).
func SnapJS(production bool) string {
	if production {
		return "https://app.midtrans.com/snap/snap.js"
	}
	return "https://app.sandbox.midtrans.com/snap/snap.js"
}

type Item struct {
	ID       string `json:"id"`
	Price    int64  `json:"price"`
	Quantity int    `json:"quantity"`
	Name     string `json:"name"`
}

type SnapRequest struct {
	OrderID   string
	Amount    int64
	Items     []Item
	Name      string
	Email     string
	FinishURL string // opsional: tujuan setelah bayar di halaman redirect
}

type SnapResponse struct {
	Token       string `json:"token"`
	RedirectURL string `json:"redirect_url"`
}

// do mengirim permintaan; kunci = Idempotency-Key ("" = tidak dikirim).
func (c *Client) do(ctx context.Context, method, url, kunci string, body any, out any) error {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rd)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.ServerKey, "")
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if kunci != "" {
		req.Header.Set("Idempotency-Key", kunci)
	}
	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 20 * time.Second}
	}
	res, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return fmt.Errorf("midtrans %s: %d %s", url, res.StatusCode, bytes.TrimSpace(raw))
	}
	return json.Unmarshal(raw, out)
}

// CreateSnap membuat transaksi Snap; token dipakai snap.pay() di browser.
func (c *Client) CreateSnap(ctx context.Context, r SnapRequest) (SnapResponse, error) {
	body := map[string]any{
		"transaction_details": map[string]any{"order_id": r.OrderID, "gross_amount": r.Amount},
		"item_details":        r.Items,
		"customer_details":    map[string]any{"first_name": r.Name, "email": r.Email},
	}
	if r.FinishURL != "" {
		body["callbacks"] = map[string]any{"finish": r.FinishURL}
	}
	var out SnapResponse
	// Idempotency-Key = order_id: retry order yang sama dalam 5 menit mendapat respons yang sama.
	// Midtrans mengabaikan kunci > 46 karakter.
	kunci := r.OrderID
	if len(kunci) > 46 {
		kunci = ""
	}
	err := c.do(ctx, http.MethodPost, c.snapURL()+"/snap/v1/transactions", kunci, body, &out)
	return out, err
}

// Notification: field notifikasi HTTP(S) Midtrans yang dipakai, juga bentuk respons Status API.
type Notification struct {
	OrderID           string `json:"order_id"`
	StatusCode        string `json:"status_code"`
	GrossAmount       string `json:"gross_amount"`
	SignatureKey      string `json:"signature_key"`
	TransactionStatus string `json:"transaction_status"`
	FraudStatus       string `json:"fraud_status"`
	PaymentType       string `json:"payment_type"`
}

// Paid: settlement, atau capture kartu yang lolos fraud check.
func (n Notification) Paid() bool {
	return n.TransactionStatus == "settlement" || (n.TransactionStatus == "capture" && n.FraudStatus == "accept")
}

// Signature: SHA512(order_id + status_code + gross_amount + server_key).
func Signature(orderID, statusCode, grossAmount, serverKey string) string {
	h := sha512.Sum512([]byte(orderID + statusCode + grossAmount + serverKey))
	return hex.EncodeToString(h[:])
}

func (c *Client) VerifySignature(n Notification) bool {
	want := Signature(n.OrderID, n.StatusCode, n.GrossAmount, c.ServerKey)
	return subtle.ConstantTimeCompare([]byte(want), []byte(n.SignatureKey)) == 1
}

// Status mengambil status transaksi langsung dari Midtrans (sumber kebenaran, bukan isi notifikasi).
func (c *Client) Status(ctx context.Context, orderID string) (Notification, error) {
	var out Notification
	err := c.do(ctx, http.MethodGet, c.apiURL()+"/v2/"+url.PathEscape(orderID)+"/status", "", nil, &out)
	return out, err
}
