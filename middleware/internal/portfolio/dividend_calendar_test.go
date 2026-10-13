package portfolio

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"investinsight/middleware/internal/marketprice"
)

func day(s string) time.Time {
	d, err := time.Parse(isoDate, s)
	if err != nil {
		panic(err)
	}
	return d
}

func dayPtr(s string) *time.Time {
	d := day(s)
	return &d
}

func eventsBy(cal *DividendCalendar, status DividendStatus) []DividendEvent {
	var out []DividendEvent
	for _, e := range cal.Events {
		if e.Status == status {
			out = append(out, e)
		}
	}
	return out
}

func TestDividendCalendarEstimatesScaleToTodaysShares(t *testing.T) {
	today := day("2026-10-13")
	held := uuid.New()
	sold := uuid.New()
	payments := []DividendPayment{
		{InstrumentID: held, Ticker: "ULVR", Date: day("2025-12-01"), AmountGBP: dec("10")},
		{InstrumentID: held, Ticker: "ULVR", Date: day("2026-06-01"), AmountGBP: dec("12")},
		{InstrumentID: sold, Ticker: "GONE", Date: day("2026-03-01"), AmountGBP: dec("5")},
	}
	// 100 shares until April, then 200.
	deltas := []ShareDelta{
		{InstrumentID: held, Date: day("2024-01-10"), SharesDelta: dec("100")},
		{InstrumentID: held, Date: day("2026-04-01"), SharesDelta: dec("100")},
	}
	holdings := []CurrentHolding{{InstrumentID: held, Ticker: "ULVR", NetShares: dec("200")}}

	cal := buildDividendCalendar(today, payments, deltas, holdings, nil)

	if got := len(eventsBy(cal, DividendPaid)); got != 3 {
		t.Errorf("paid events = %d, want 3 (sold holdings still count as paid)", got)
	}
	est := eventsBy(cal, DividendEstimated)
	if len(est) != 2 {
		t.Fatalf("estimated events = %+v, want 2 (none for the sold holding)", est)
	}
	// Dec: paid on 100 shares, now 200 → doubled. Jun: paid on 200 → same.
	if est[0].Date != "2026-12-01" || !est[0].AmountGBP.Equal(dec("20")) {
		t.Errorf("first estimate = %s £%v, want 2026-12-01 £20", est[0].Date, est[0].AmountGBP)
	}
	if est[1].Date != "2027-06-01" || !est[1].AmountGBP.Equal(dec("12")) {
		t.Errorf("second estimate = %s £%v, want 2027-06-01 £12", est[1].Date, est[1].AmountGBP)
	}
	if cal.From != "2025-10-13" || cal.To != "2027-10-13" {
		t.Errorf("window = %s..%s", cal.From, cal.To)
	}
}

func TestDividendCalendarAnnouncementReplacesNearestEstimate(t *testing.T) {
	today := day("2026-10-13")
	id := uuid.New()
	payments := []DividendPayment{
		{InstrumentID: id, Ticker: "AAPL", Date: day("2025-11-14"), AmountGBP: dec("3")},
		{InstrumentID: id, Ticker: "AAPL", Date: day("2026-02-13"), AmountGBP: dec("3")},
	}
	deltas := []ShareDelta{{InstrumentID: id, Date: day("2024-01-01"), SharesDelta: dec("10")}}
	holdings := []CurrentHolding{{InstrumentID: id, Ticker: "AAPL", NetShares: dec("10")}}
	announced := map[uuid.UUID]marketprice.DividendDates{
		id: {ExDate: dayPtr("2026-11-10"), PayDate: dayPtr("2026-11-13")},
	}

	cal := buildDividendCalendar(today, payments, deltas, holdings, announced)

	ann := eventsBy(cal, DividendAnnounced)
	if len(ann) != 1 {
		t.Fatalf("announced = %+v, want 1", ann)
	}
	if ann[0].Date != "2026-11-13" || *ann[0].ExDate != "2026-11-10" || !ann[0].AmountGBP.Equal(dec("3")) {
		t.Errorf("announced = %s ex %v £%v, want 2026-11-13 ex 2026-11-10 £3", ann[0].Date, *ann[0].ExDate, ann[0].AmountGBP)
	}
	if est := eventsBy(cal, DividendEstimated); len(est) != 1 || est[0].Date != "2027-02-13" {
		t.Errorf("remaining estimates = %+v, want only 2027-02-13", est)
	}
}

func TestDividendCalendarAnnouncementWithoutHistory(t *testing.T) {
	today := day("2026-10-13")
	id := uuid.New()
	holdings := []CurrentHolding{{InstrumentID: id, Ticker: "NEW", NetShares: dec("5")}}
	announced := map[uuid.UUID]marketprice.DividendDates{id: {ExDate: dayPtr("2026-10-30")}}

	cal := buildDividendCalendar(today, nil, nil, holdings, announced)

	if len(cal.Events) != 1 || cal.Events[0].Status != DividendAnnounced || cal.Events[0].AmountGBP != nil {
		t.Fatalf("events = %+v, want one announced event with no amount", cal.Events)
	}
	if cal.Events[0].Date != "2026-10-30" {
		t.Errorf("date = %s, want the ex-date when no payment date is known", cal.Events[0].Date)
	}
}

func TestDividendCalendarSkipsEstimatesOutsideTheYear(t *testing.T) {
	today := day("2026-10-13")
	id := uuid.New()
	// Paid exactly a year ago: a year on is today, not upcoming.
	payments := []DividendPayment{{InstrumentID: id, Ticker: "X", Date: day("2025-10-13"), AmountGBP: dec("1")}}
	deltas := []ShareDelta{{InstrumentID: id, Date: day("2024-01-01"), SharesDelta: dec("1")}}
	holdings := []CurrentHolding{{InstrumentID: id, Ticker: "X", NetShares: dec("1")}}

	cal := buildDividendCalendar(today, payments, deltas, holdings, nil)

	if est := eventsBy(cal, DividendEstimated); len(est) != 0 {
		t.Errorf("estimates = %+v, want none", est)
	}
	if cal.Events == nil {
		t.Error("events is nil, want an empty slice")
	}
}
