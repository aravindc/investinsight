package trading212

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

func testCipher(t *testing.T) *Cipher {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	c, err := NewCipher(base64.StdEncoding.EncodeToString(key))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestCipher_RoundTripAndAccountBinding(t *testing.T) {
	c := testCipher(t)
	enc, err := c.Encrypt([]byte("secret-value"), []byte("account-a"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.Decrypt(enc, []byte("account-a"))
	if err != nil || string(got) != "secret-value" {
		t.Fatalf("Decrypt = %q, %v", got, err)
	}
	if _, err := c.Decrypt(enc, []byte("account-b")); err == nil {
		t.Fatal("ciphertext decrypted under a different account id")
	}
	enc2, _ := c.Encrypt([]byte("secret-value"), []byte("account-a"))
	if string(enc) == string(enc2) {
		t.Fatal("encrypting twice produced identical ciphertext (nonce reuse)")
	}
}

func TestCipher_Config(t *testing.T) {
	if c, err := NewCipher(""); c != nil || err != nil {
		t.Fatalf("empty key: got %v, %v; want nil, nil", c, err)
	}
	if _, err := NewCipher(base64.StdEncoding.EncodeToString([]byte("too short"))); err == nil {
		t.Fatal("short key accepted")
	}
	var disabled *Cipher
	if _, err := disabled.Encrypt([]byte("x"), nil); err != ErrNoEncryptionKey {
		t.Fatalf("nil cipher Encrypt err = %v", err)
	}
}

func TestExportWindows(t *testing.T) {
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(2*exportWindow + time.Hour)
	ws := exportWindows(from, to, exportWindow)
	if len(ws) != 3 {
		t.Fatalf("got %d windows, want 3", len(ws))
	}
	if !ws[0].from.Equal(from) || !ws[2].to.Equal(to) {
		t.Fatalf("windows don't span [from, to): %+v", ws)
	}
	for i := 1; i < len(ws); i++ {
		if !ws[i].from.Equal(ws[i-1].to) {
			t.Fatalf("gap between windows %d and %d", i-1, i)
		}
	}
	if len(exportWindows(to, to, exportWindow)) != 0 {
		t.Fatal("empty range produced windows")
	}
}

// fakeT212 serves the account summary and export endpoints. The report is
// "Queued" on the first status poll and "Finished" afterwards.
func fakeT212(t *testing.T, rateLimitFirstRequest bool) (*httptest.Server, *int32) {
	t.Helper()
	var exportRequests, listCalls int32
	mux := http.NewServeMux()
	var srv *httptest.Server

	auth := func(w http.ResponseWriter, r *http.Request) bool {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "key-1234" || pass != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return false
		}
		return true
	}

	mux.HandleFunc("GET /api/v0/equity/account/summary", func(w http.ResponseWriter, r *http.Request) {
		if auth(w, r) {
			json.NewEncoder(w).Encode(map[string]any{"id": 42, "currency": "GBP"})
		}
	})
	mux.HandleFunc("POST /api/v0/equity/history/exports", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		if atomic.AddInt32(&exportRequests, 1) == 1 && rateLimitFirstRequest {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		var body struct {
			TimeFrom string `json:"timeFrom"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if _, err := time.Parse(time.RFC3339, body.TimeFrom); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"reportId": 7})
	})
	mux.HandleFunc("GET /api/v0/equity/history/exports", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		status, link := "Queued", ""
		if atomic.AddInt32(&listCalls, 1) > 1 {
			status, link = "Finished", srv.URL+"/download/7.csv"
		}
		json.NewEncoder(w).Encode([]map[string]any{
			{"reportId": 6, "status": "Finished", "downloadLink": srv.URL + "/download/6.csv"},
			{"reportId": 7, "status": status, "downloadLink": link},
		})
	})
	mux.HandleFunc("GET /download/7.csv", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("credentials were sent to the download link")
		}
		w.Write([]byte("Action,Time (UTC)\n"))
	})

	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &exportRequests
}

func TestClient_AccountSummary(t *testing.T) {
	srv, _ := fakeT212(t, false)
	client, _ := NewClient(srv.URL)

	summary, err := client.AccountSummary(context.Background(), Credentials{"key-1234", "secret"})
	if err != nil || summary.ID != 42 || summary.Currency != "GBP" {
		t.Fatalf("AccountSummary = %+v, %v", summary, err)
	}
	if _, err := client.AccountSummary(context.Background(), Credentials{"key-1234", "wrong"}); err != ErrUnauthorized {
		t.Fatalf("bad secret err = %v, want ErrUnauthorized", err)
	}
}

func TestService_ExportWindow(t *testing.T) {
	srv, exportRequests := fakeT212(t, true)
	client, _ := NewClient(srv.URL)
	s := NewService(nil, client, nil, nil, nil, zap.NewNop())
	s.requestExportInterval = time.Millisecond
	s.listExportsInterval = time.Millisecond

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	csv, err := s.exportWindow(context.Background(), &rateLimiter{}, Credentials{"key-1234", "secret"},
		window{from, from.Add(24 * time.Hour)})
	if err != nil {
		t.Fatalf("exportWindow: %v", err)
	}
	if string(csv) != "Action,Time (UTC)\n" {
		t.Fatalf("downloaded %q", csv)
	}
	if n := atomic.LoadInt32(exportRequests); n != 2 {
		t.Fatalf("export requested %d times, want 2 (one 429 retry)", n)
	}
}
