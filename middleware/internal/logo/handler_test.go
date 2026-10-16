package logo

import "testing"

func TestLogoETag(t *testing.T) {
	a, b := logoETag([]byte("placeholder")), logoETag([]byte("real logo"))
	if a == b {
		t.Error("different images share an ETag")
	}
	if a != logoETag([]byte("placeholder")) {
		t.Error("ETag isn't stable for the same image")
	}
	if a[0] != '"' || a[len(a)-1] != '"' {
		t.Errorf("ETag %s isn't quoted", a)
	}
}

func TestETagMatches(t *testing.T) {
	etag := logoETag([]byte("logo"))
	cases := map[string]bool{
		"":                      false,
		etag:                    true,
		"W/" + etag:             true,
		`"other", ` + etag:      true,
		"*":                     true,
		logoETag([]byte("old")): false,
	}
	for header, want := range cases {
		if got := etagMatches(header, etag); got != want {
			t.Errorf("etagMatches(%q) = %v, want %v", header, got, want)
		}
	}
}
