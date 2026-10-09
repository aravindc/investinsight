package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCrossOriginProtection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(crossOriginProtection())
	router.GET("/api/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.POST("/api/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	tests := []struct {
		name    string
		method  string
		headers map[string]string
		want    int
	}{
		{"same-origin fetch", http.MethodPost, map[string]string{"Sec-Fetch-Site": "same-origin", "Origin": "https://invest.example.com"}, http.StatusOK},
		{"cross-site fetch", http.MethodPost, map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://evil.example"}, http.StatusForbidden},
		{"same-site subdomain", http.MethodPost, map[string]string{"Sec-Fetch-Site": "same-site", "Origin": "https://other.example.com"}, http.StatusForbidden},
		{"older browser, matching Origin", http.MethodPost, map[string]string{"Origin": "https://invest.example.com"}, http.StatusOK},
		{"older browser, foreign Origin", http.MethodPost, map[string]string{"Origin": "https://evil.example"}, http.StatusForbidden},
		{"non-browser client", http.MethodPost, nil, http.StatusOK},
		{"cross-site read", http.MethodGet, map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://evil.example"}, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "https://invest.example.com/api/x", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Errorf("got %d, want %d", rec.Code, tt.want)
			}
		})
	}
}
