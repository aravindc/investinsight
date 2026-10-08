package trading212

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestParseCashScope(t *testing.T) {
	id := uuid.New()
	if s, err := ParseCashScope(id.String()); err != nil || s.Household || s.AccountID != id {
		t.Errorf("account scope = %+v, %v", s, err)
	}
	if s, err := ParseCashScope("household:" + id.String()); err != nil || !s.Household || s.HouseholdID != id {
		t.Errorf("household scope = %+v, %v", s, err)
	}
	for _, bad := range []string{"", "nope", "household:nope"} {
		if _, err := ParseCashScope(bad); err == nil {
			t.Errorf("ParseCashScope(%q) accepted", bad)
		}
	}
}

func TestSummaryCache_MaxAge(t *testing.T) {
	var c summaryCache
	id := uuid.New()
	if _, ok := c.get(id, time.Minute); ok {
		t.Fatal("empty cache hit")
	}
	c.set(id, &AccountSummary{ID: 1})
	if s, ok := c.get(id, time.Minute); !ok || s.ID != 1 {
		t.Fatal("fresh entry missed")
	}
	time.Sleep(5 * time.Millisecond)
	if _, ok := c.get(id, time.Millisecond); ok {
		t.Fatal("stale entry returned")
	}
	c.forget(id)
	if _, ok := c.get(id, time.Hour); ok {
		t.Fatal("forgotten entry returned")
	}
}
