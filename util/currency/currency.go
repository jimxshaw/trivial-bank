package currency

// Constants for all supported currencies.
const (
	USD = "USD"
	EUR = "EUR"
	GBP = "GBP"
	CAD = "CAD"
	CNY = "CNY"
	AUD = "AUD"
	MXN = "MXN"
)

func IsSupportedCurrency(currency string) bool {
	switch currency {
	case USD, EUR, GBP, CAD, CNY, AUD, MXN:
		return true
	}
	return false
}
