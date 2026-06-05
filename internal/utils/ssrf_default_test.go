package utils

import "testing"

// Locks in the fail-closed default: strict SSRF protection is ON unless the
// operator explicitly opts out with WEBHOOK_STRICT_SSRF=false.
func TestWebhookStrictSSRFEnabled(t *testing.T) {
	cases := []struct {
		val  string
		set  bool
		want bool
	}{
		{"", false, true},      // unset -> on (fail-closed)
		{"true", true, true},   // explicit on
		{"false", true, false}, // explicit opt-out
		{"1", true, true},      // anything other than "false" -> on
	}
	for _, c := range cases {
		if c.set {
			t.Setenv("WEBHOOK_STRICT_SSRF", c.val)
		} else {
			t.Setenv("WEBHOOK_STRICT_SSRF", "")
			// t.Setenv("","") leaves it set to empty; emulate unset via empty value,
			// which WebhookStrictSSRFEnabled treats as "not false" -> enabled.
		}
		if got := WebhookStrictSSRFEnabled(); got != c.want {
			t.Errorf("WEBHOOK_STRICT_SSRF=%q -> %v, want %v", c.val, got, c.want)
		}
	}
}
