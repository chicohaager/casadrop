package email

import "testing"

// stripHeaderValue must remove CR/LF so a user-controlled value (e.g. the email
// subject, derived from an attacker-controlled transfer title/sender) can't
// inject extra MIME headers or a forged body into the outbound message.
func TestStripHeaderValue(t *testing.T) {
	cases := map[string]string{
		"normal subject":                    "normal subject",
		"evil\r\nBcc: attacker@example.com": "evilBcc: attacker@example.com",
		"line1\nline2":                      "line1line2",
		"carriage\rreturn":                  "carriagereturn",
		"\r\n\r\ninjected body":             "injected body",
	}
	for in, want := range cases {
		if got := stripHeaderValue(in); got != want {
			t.Errorf("stripHeaderValue(%q) = %q, want %q", in, got, want)
		}
		if got := stripHeaderValue(in); containsCRLF(got) {
			t.Errorf("stripHeaderValue(%q) still contains CR/LF: %q", in, got)
		}
	}
}

func containsCRLF(s string) bool {
	for _, r := range s {
		if r == '\r' || r == '\n' {
			return true
		}
	}
	return false
}

// The share dialog sends no sender, so an empty sender must fall back to the
// configured SMTP identity rather than producing an empty "shared a file with
// you" line. Regression for the "sender_email is required" bug that made every
// email-share fail even with correct SMTP settings.
func TestApplySenderFallback(t *testing.T) {
	cfg := SMTPConfigForTest("from@example.com", "CasaDrop")
	s := NewService(&cfg)

	// Empty sender: both fields fall back to the SMTP identity.
	tr := &transferForTest{}
	s.applySenderFallback(tr.e())
	if tr.msg.SenderEmail != "from@example.com" || tr.msg.SenderName != "CasaDrop" {
		t.Errorf("empty sender not filled from config: %+v", tr.msg)
	}

	// A caller-supplied sender is preserved.
	tr2 := &transferForTest{email: "user@example.com", name: "Alice"}
	m := tr2.e()
	s.applySenderFallback(m)
	if m.SenderEmail != "user@example.com" || m.SenderName != "Alice" {
		t.Errorf("caller sender overwritten: %+v", m)
	}
}
