package email

import (
	"mime"
	"strings"
	"testing"
	"time"

	"casadrop/internal/models"
)

// E-mails go out in the sender's language (2.5.2). The values that come from
// users (file name, address) must stay escaped inside the translated sentence.
func TestDownloadNotificationLanguageAndEscaping(t *testing.T) {
	s := &Service{}
	html := s.buildDownloadNotificationHTML("de", "Anna", "gast@example.org", `<script>x</script>.pdf`)
	for _, want := range []string{`lang="de"`, "Deine Datei wurde heruntergeladen!", "heruntergeladen", "Gesendet mit CasaDrop"} {
		if !strings.Contains(html, want) {
			t.Errorf("German download mail lacks %q", want)
		}
	}
	if strings.Contains(html, "<script>") {
		t.Error("file name not escaped inside the translated sentence")
	}
	if !strings.Contains(html, "&lt;script&gt;x&lt;/script&gt;.pdf") {
		t.Error("escaped file name missing")
	}
	if strings.Contains(html, "{file}") || strings.Contains(html, "{recipient}") {
		t.Error("unfilled placeholder in the mail")
	}
}

func TestTransferMailInSenderLanguage(t *testing.T) {
	s := &Service{}
	tr := testTransfer("ja")
	html, err := s.buildTransferEmailHTML(tr, "https://drop.example.org/s/abc", "a.pdf", "1 KB")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `lang="ja"`) || strings.Contains(html, "shared a file with you") {
		t.Errorf("Japanese transfer mail still English or wrong lang attribute")
	}
}

// A transfer stored before 2.5.2 has no language: English, never a raw key.
func TestOldRecordWithoutLanguageFallsBackToEnglish(t *testing.T) {
	s := &Service{}
	html := s.buildDownloadNotificationHTML("", "Anna", "gast@example.org", "a.pdf")
	if !strings.Contains(html, "Your file was downloaded!") || !strings.Contains(html, `lang="en"`) {
		t.Error("empty language did not fall back to English")
	}
	if strings.Contains(html, "email.download") {
		t.Error("raw i18n key in mail")
	}
}

func TestExpiryWarningLocalized(t *testing.T) {
	when := time.Date(2026, 10, 5, 14, 15, 0, 0, time.UTC)
	html := buildExpiryWarningHTML("de", "gast@example.org", "", "a<b>.pdf", when)
	for _, want := range []string{`lang="de"`, "Datei läuft bald ab", "Hallo gast@example.org,", "05.10.2026, 14:15", "a&lt;b&gt;.pdf"} {
		if !strings.Contains(html, want) {
			t.Errorf("German expiry mail lacks %q", want)
		}
	}
	if strings.Contains(html, "{date}") || strings.Contains(html, "a<b>") {
		t.Error("placeholder left or file name unescaped")
	}
	if en := buildExpiryWarningHTML("en", "x@example.org", "", "a.pdf", when); !strings.Contains(en, "Oct 5, 2026, 2:15 PM") {
		t.Error("English date format not applied")
	}
}

func testTransfer(lang string) *models.EmailTransfer {
	return &models.EmailTransfer{SenderName: "Anna", SenderEmail: "anna@example.org", Lang: lang}
}

// Since 2.5.2 the default subject is in the sender's language. Raw 8-bit
// header bytes get mangled by MTAs without SMTPUTF8, so non-ASCII subjects and
// sender names must be RFC 2047-encoded — and decode back to the original.
func TestHeadersAreRFC2047Encoded(t *testing.T) {
	s := &Service{config: &models.SMTPConfig{FromEmail: "drop@example.org", FromName: "Café Drop"}}
	subject := "Datei läuft bald ab: Ärger.pdf"
	msg := string(s.buildMessage("to@example.org", subject, "<p>x</p>"))
	head := msg[:strings.Index(msg, "\r\n\r\n")]
	for i := 0; i < len(head); i++ {
		if head[i] > 127 {
			t.Fatalf("raw non-ASCII byte in headers: %q", head)
		}
	}
	var gotSubject, gotFrom string
	for _, line := range strings.Split(head, "\r\n") {
		if v, ok := strings.CutPrefix(line, "Subject: "); ok {
			gotSubject = v
		}
		if v, ok := strings.CutPrefix(line, "From: "); ok {
			gotFrom = v
		}
	}
	dec := new(mime.WordDecoder)
	if d, err := dec.DecodeHeader(gotSubject); err != nil || d != subject {
		t.Errorf("subject decodes to %q (%v), want %q", d, err, subject)
	}
	if d, err := dec.DecodeHeader(gotFrom); err != nil || !strings.Contains(d, "Café Drop") {
		t.Errorf("from decodes to %q (%v)", d, err)
	}
	if strings.Contains(head, "\r\nX-Injected") {
		t.Error("header injection")
	}
}

// A multi-line message used to show a literal "<br>" in the mail; it must be
// a real line break and the message text itself stays escaped.
func TestMessageLineBreaksAndEscaping(t *testing.T) {
	s := &Service{}
	tr := testTransfer("en")
	tr.Message = "line one\n<b>line two</b>"
	html, err := s.buildTransferEmailHTML(tr, "https://drop.example.org/s/abc", "a.pdf", "1 KB")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, "&lt;br&gt;") {
		t.Error("line break rendered as literal <br>")
	}
	if !strings.Contains(html, "line one<br>&lt;b&gt;line two&lt;/b&gt;") {
		t.Error("message not escaped or line break missing")
	}
}
