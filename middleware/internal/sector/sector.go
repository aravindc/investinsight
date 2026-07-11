// Package sector provides a static, hand-curated ticker-to-sector mapping
// used to categorize holdings for allocation reporting. There is no sector
// data anywhere upstream (broker CSVs, the instruments table, or the Yahoo
// chart API already in use), so this is maintained by hand rather than
// fetched live.
package sector

import "strings"

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

// underlyingTicker maps single-stock options-income wrapper products (e.g.
// IncomeShares, YieldMax) to the ticker of the stock they derive income
// from, so they're categorized by that stock's sector rather than treated
// as their own asset class.
var underlyingTicker = map[string]string{
	"AAPI": "AAPL",
	"AMDI": "AMD",
	"AMZD": "AMZN",
	"AVGI": "AVGO",
	"COII": "COIN",
	"GOOO": "GOOGL",
	"METI": "META",
	"MSFI": "MSFT",
	"MSTP": "MSTR",
	"NVDD": "NVDA",
	"TSLD": "TSLA",
	"MAGD": "", // Magnificent 7 basket — not a single underlying
	"QQQO": "", // Nasdaq 100 basket
	"SPYO": "", // S&P 500 basket
	"XYLP": "", // S&P 500 covered call basket
	"QYLP": "", // Nasdaq 100 covered call basket
	"TLTI": "", // 20+ Year Treasury — bonds, not equity
	"GLDE": "", // Gold — commodity
	"SLVI": "", // Silver — commodity
}

// bySector maps each ticker (or underlying ticker for wrapper products) to
// its sector. Tickers not present here fall back to FundsAndETFs for known
// fund/ETF wrappers, or Other.
var bySector = map[string]string{
	// Technology
	"AAPL": Technology,
	"AMD":  Technology,
	"AVGO": Technology,
	"MSFT": Technology,
	"NVDA": Technology,
	"SGE":  Technology,
	"BKSY": Technology,
	"PL":   Technology,
	"SMX":  Technology,
	"SPCX": Technology,

	// Communication Services
	"GOOGL": CommunicationServices,
	"META":  CommunicationServices,

	// Consumer Discretionary
	"AMZN": ConsumerDiscretionary,
	"TSLA": ConsumerDiscretionary,
	"MKS":  ConsumerDiscretionary,
	"TSCO": ConsumerDiscretionary,
	"SBRY": ConsumerDiscretionary,
	"PSN":  ConsumerDiscretionary,
	"TW":   ConsumerDiscretionary,

	// Consumer Staples
	"KO":   ConsumerStaples,
	"PG":   ConsumerStaples,
	"DGE":  ConsumerStaples,
	"BATS": ConsumerStaples,
	"IMB":  ConsumerStaples,
	"ABF":  ConsumerStaples,
	"CCH":  ConsumerStaples,
	"BAG":  ConsumerStaples,
	"BVIC": ConsumerStaples,
	"TATE": ConsumerStaples,
	"ULVR": ConsumerStaples,
	"MICC": ConsumerStaples,

	// Healthcare
	"AZN": Healthcare,
	"GSK": Healthcare,

	// Financials
	"AV":   Financials,
	"ABDN": Financials,
	"BARC": Financials,
	"HSBA": Financials,
	"LGEN": Financials,
	"LLOY": Financials,
	"MNG":  Financials,
	"MONY": Financials,
	"NWG":  Financials,
	"SDLF": Financials,
	"MAIN": Financials,
	"OXLC": Financials,
	"DBK":  Financials,
	"COIN": Financials,

	// Energy
	"BP":   Energy,
	"SHEL": Energy,

	// Materials
	"AAL":  Materials,
	"RIO":  Materials,
	"GLEN": Materials,
	"CEY":  Materials,
	"AU":   Materials,
	"VALT": Materials,
	"MNDI": Materials,

	// Industrials
	"BA":   Industrials,
	"BNZL": Industrials,

	// Utilities
	"NG":   Utilities,
	"SSE":  Utilities,
	"UU":   Utilities,
	"NESF": Utilities,
	"TRIG": Utilities,
	"UKW":  Utilities,

	// Real Estate
	"AGR":  RealEstate,
	"BLND": RealEstate,
	"BYG":  RealEstate,
	"LAND": RealEstate,
	"PHP":  RealEstate,
	"SGRO": RealEstate,
	"O":    RealEstate,

	// Consumer Discretionary (autos)
	"BMW": ConsumerDiscretionary,

	// Funds & ETFs (multi-sector baskets)
	"EQQQ": FundsAndETFs,
	"IIND": FundsAndETFs,
	"INCU": FundsAndETFs,
	"JEGP": FundsAndETFs,
	"JEIP": FundsAndETFs,
	"JEQP": FundsAndETFs,
	"JGGI": FundsAndETFs,
	"SMIF": FundsAndETFs,
	"SSHY": FundsAndETFs,
	"STHS": FundsAndETFs,
	"VAGS": FundsAndETFs,
	"VHYL": FundsAndETFs,
	"VPNG": FundsAndETFs,
	"VUAG": FundsAndETFs,
	"VUSA": FundsAndETFs,
	"VWRP": FundsAndETFs,
	"WINC": FundsAndETFs,
	"CEPG": FundsAndETFs,
	"FEPD": FundsAndETFs,
	"FEPG": FundsAndETFs,
	"MAGD": FundsAndETFs,
	"QQQO": FundsAndETFs,
	"SPYO": FundsAndETFs,
	"XYLP": FundsAndETFs,
	"QYLP": FundsAndETFs,
	"TLTI": FundsAndETFs,
	"GLDE": FundsAndETFs,
	"SLVI": FundsAndETFs,
	"YMAP": FundsAndETFs,
}

// ForTicker returns the sector for a given ticker. Single-stock
// options-income wrapper products resolve to their underlying's sector
// first. Unrecognized tickers return Other.
func ForTicker(ticker string) string {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))

	if underlying, ok := underlyingTicker[ticker]; ok && underlying != "" {
		ticker = underlying
	}

	if s, ok := bySector[ticker]; ok {
		return s
	}
	return Other
}
