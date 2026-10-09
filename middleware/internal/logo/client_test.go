package logo

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("dial tcp: connection refused")
}

func TestFetchLogoErrorOmitsToken(t *testing.T) {
	const token = "pk_secret_token_value"
	c := NewClient(token)
	c.httpClient.Transport = failingTransport{}

	_, err := c.FetchLogo(context.Background(), "LLOY.L")
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("error leaks the token: %v", err)
	}
	if !strings.Contains(err.Error(), "img.logo.dev/ticker/LLOY.L") {
		t.Errorf("error should still name the request: %v", err)
	}
}
