// Package sector defines the fixed set of sector names instruments can be
// classified into. Sector is stored per-instrument in the database
// (instruments.sector, editable via the Admin page) — this package no
// longer does any ticker-to-sector lookup itself; it only defines the
// valid set for server-side validation when an Admin edit comes in.
package sector

const (
	Technology            = "Technology"
	Financials            = "Financials"
	Healthcare            = "Healthcare"
	ConsumerStaples       = "Consumer Staples"
	ConsumerDiscretionary = "Consumer Discretionary"
	Energy                = "Energy"
	Materials             = "Materials"
	Industrials           = "Industrials"
	Utilities             = "Utilities"
	RealEstate            = "Real Estate"
	CommunicationServices = "Communication Services"
	FundsAndETFs          = "Funds & ETFs"
	Other                 = "Other"
)

// All lists every valid sector, in the fixed display order used
// throughout the app (matching frontend/src/lib/sector.ts's SECTOR_ORDER).
func All() []string {
	return []string{
		Technology,
		Financials,
		Healthcare,
		ConsumerStaples,
		ConsumerDiscretionary,
		Energy,
		Materials,
		Industrials,
		Utilities,
		RealEstate,
		CommunicationServices,
		FundsAndETFs,
		Other,
	}
}

// IsValid reports whether s is one of the known sector names.
func IsValid(s string) bool {
	for _, valid := range All() {
		if s == valid {
			return true
		}
	}
	return false
}
