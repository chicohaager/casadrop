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
