package sector

import (
	"strings"
	"testing"
)

func TestNormalizeName(t *testing.T) {
	cases := map[string]string{
		"Defence":                  "Defence",
		"  Aerospace   & Defence ": "Aerospace & Defence",
	}
	for in, want := range cases {
		if got, err := NormalizeName(in); err != nil || got != want {
			t.Errorf("NormalizeName(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "   ", strings.Repeat("x", 51), "Oil/Gas"} {
		if _, err := NormalizeName(bad); err == nil {
			t.Errorf("NormalizeName(%q) accepted", bad)
		}
	}
	if _, err := NormalizeName(strings.Repeat("é", 50)); err != nil {
		t.Errorf("50 multi-byte characters rejected: %v", err)
	}
}
