package handlers

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"casadrop/internal/models"
	"casadrop/internal/storage"
)

// fakeSMTP is a minimal SMTP server: it accepts mail, or rejects every
// recipient when reject is set, and counts delivered messages.
type fakeSMTP struct {
	addr      *net.TCPAddr
	delivered atomic.Int32
}

func startFakeSMTP(t *testing.T, reject bool) *fakeSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	f := &fakeSMTP{addr: ln.Addr().(*net.TCPAddr)}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go f.serve(c, reject)
		}
	}()
	return f
}

func (f *fakeSMTP) serve(c net.Conn, reject bool) {
	defer c.Close()
	r := bufio.NewReader(c)
	say := func(s string) { c.Write([]byte(s + "\r\n")) }
	say("220 fake ESMTP")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			say("250 fake")
		case strings.HasPrefix(cmd, "RCPT") && reject:
			say("550 mailbox unavailable")
		case strings.HasPrefix(cmd, "MAIL"), strings.HasPrefix(cmd, "RCPT"), strings.HasPrefix(cmd, "RSET"), strings.HasPrefix(cmd, "NOOP"):
			say("250 OK")
		case cmd == "DATA":
			say("354 go ahead")
			for {
				l, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if l == ".\r\n" {
					break
				}
			}
			f.delivered.Add(1)
			say("250 queued")
		case cmd == "QUIT":
			say("221 bye")
			return
		default:
			say("502 unknown")
		}
	}
}

func emailHandlerWithSMTP(t *testing.T, smtp *fakeSMTP) (*EmailHandler, *storage.Storage) {
	t.Helper()
	store := newEmailTestStore(t)
	if err := store.SaveSMTPConfig(&models.SMTPConfig{Enabled: true, Host: "127.0.0.1", Port: smtp.addr.Port,
		FromEmail: "drop@example.org", FromName: "CasaDrop"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(&models.Share{ID: "share-1", FileName: "doc.pdf", OriginalName: "doc.pdf",
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	return NewEmailHandler(store), store
}

func sendTransfer(h *EmailHandler) *httptest.ResponseRecorder {
	body := `{"share_id":"share-1","recipient_email":"someone@example.org","lang":"fr"}`
	rr := httptest.NewRecorder()
	h.SendEmailTransfer(rr, httptest.NewRequest("POST", "/api/email/send", strings.NewReader(body)))
	return rr
}

func transferRecords(t *testing.T, store *storage.Storage) []*models.EmailTransferRecord {
	t.Helper()
	recs, err := store.GetEmailTransfersByShare("share-1")
	if err != nil {
		t.Fatal(err)
	}
	return recs
}

func TestSentTransferIsRecorded(t *testing.T) {
	smtp := startFakeSMTP(t, false)
	h, store := emailHandlerWithSMTP(t, smtp)
	if rr := sendTransfer(h); rr.Code != 200 {
		t.Fatalf("send: %d %s", rr.Code, rr.Body.String())
	}
	if n := smtp.delivered.Load(); n != 1 {
		t.Errorf("%d mails delivered, want 1", n)
	}
	recs := transferRecords(t, store)
	if len(recs) != 1 || recs[0].Lang != "fr" {
		t.Fatalf("want one record with lang fr, got %d", len(recs))
	}
}

// 2.5.2: the record is saved before the mail goes out. If it cannot be saved,
// no mail is sent and the sender gets an error — previously the mail went out,
// the failure was only logged, and the download/expiry mails never came.
func TestNoMailWithoutRecord(t *testing.T) {
	smtp := startFakeSMTP(t, false)
	h, store := emailHandlerWithSMTP(t, smtp)
	if err := store.DropEmailTransfersTableForTest(); err != nil {
		t.Fatal(err)
	}
	rr := sendTransfer(h)
	if rr.Code != 500 {
		t.Fatalf("got %d, want 500 when the record cannot be saved", rr.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil || !strings.Contains(resp["error"], "not sent") {
		t.Errorf("error response does not say the mail was not sent: %s", rr.Body.String())
	}
	if n := smtp.delivered.Load(); n != 0 {
		t.Errorf("%d mails delivered although the record was not saved", n)
	}
}

// If the mail cannot be sent, its record is removed again — otherwise the
// recipient would later get an expiry warning for a share they never got.
func TestUnsentTransferLeavesNoRecord(t *testing.T) {
	smtp := startFakeSMTP(t, true)
	h, store := emailHandlerWithSMTP(t, smtp)
	if rr := sendTransfer(h); rr.Code != 500 {
		t.Fatalf("got %d, want 500 when SMTP rejects the recipient", rr.Code)
	}
	if recs := transferRecords(t, store); len(recs) != 0 {
		t.Errorf("%d record(s) left for a mail that was never sent", len(recs))
	}
}
