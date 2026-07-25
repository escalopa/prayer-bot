package location

import "strings"

// currencyByCountry maps ISO 3166-1 alpha-2 country codes to their ISO 4217
// currency code. It is used only to choose a sensible default currency for the
// Zakat calculator; the user can always override the selection. Coverage favors
// the bot's audience (Muslim-majority countries) and major economies; unlisted
// countries fall back to USD at the call site.
var currencyByCountry = map[string]string{
	// Middle East & North Africa
	"SA": "SAR", "AE": "AED", "EG": "EGP", "QA": "QAR", "KW": "KWD",
	"BH": "BHD", "OM": "OMR", "JO": "JOD", "LB": "LBP", "IQ": "IQD",
	"SY": "SYP", "YE": "YER", "PS": "ILS", "IL": "ILS", "DZ": "DZD",
	"MA": "MAD", "TN": "TND", "LY": "LYD", "SD": "SDG", "MR": "MRU",
	// Türkiye, Iran, South & Central Asia
	"TR": "TRY", "IR": "IRR", "PK": "PKR", "IN": "INR", "BD": "BDT",
	"AF": "AFN", "LK": "LKR", "MV": "MVR", "NP": "NPR",
	"UZ": "UZS", "KZ": "KZT", "KG": "KGS", "TJ": "TJS", "TM": "TMT",
	"AZ": "AZN", "GE": "GEL", "AM": "AMD",
	// Southeast & East Asia
	"ID": "IDR", "MY": "MYR", "BN": "BND", "SG": "SGD", "TH": "THB",
	"PH": "PHP", "VN": "VND", "CN": "CNY", "JP": "JPY", "KR": "KRW",
	"HK": "HKD", "TW": "TWD",
	// Sub-Saharan Africa
	"NG": "NGN", "SO": "SOS", "DJ": "DJF", "KM": "KMF", "SN": "XOF",
	"ML": "XOF", "NE": "XOF", "BF": "XOF", "CI": "XOF", "TD": "XAF",
	"CM": "XAF", "GN": "GNF", "GM": "GMD", "SL": "SLE", "KE": "KES",
	"TZ": "TZS", "UG": "UGX", "ET": "ETB", "ZA": "ZAR", "GH": "GHS",
	// Europe (euro area)
	"DE": "EUR", "FR": "EUR", "ES": "EUR", "IT": "EUR", "NL": "EUR",
	"BE": "EUR", "AT": "EUR", "PT": "EUR", "IE": "EUR", "FI": "EUR",
	"GR": "EUR", "SK": "EUR", "SI": "EUR", "LT": "EUR", "LV": "EUR",
	"EE": "EUR", "LU": "EUR", "CY": "EUR", "MT": "EUR", "HR": "EUR",
	// Europe (non-euro) & CIS
	"GB": "GBP", "CH": "CHF", "SE": "SEK", "NO": "NOK", "DK": "DKK",
	"PL": "PLN", "CZ": "CZK", "HU": "HUF", "RO": "RON", "BG": "BGN",
	"RS": "RSD", "UA": "UAH", "RU": "RUB", "BY": "BYN", "AL": "ALL",
	"BA": "BAM", "MK": "MKD", "MD": "MDL",
	// Americas & Oceania
	"US": "USD", "CA": "CAD", "MX": "MXN", "BR": "BRL", "AR": "ARS",
	"CL": "CLP", "CO": "COP", "PE": "PEN", "AU": "AUD", "NZ": "NZD",
}

// timezoneCountry maps common IANA timezones to a country code. It is a
// best-effort fallback for profiles saved before country codes were persisted,
// so it only needs to cover frequently used zones; anything unlisted falls back
// to USD at the call site.
var timezoneCountry = map[string]string{
	"Africa/Cairo": "EG", "Africa/Algiers": "DZ", "Africa/Casablanca": "MA",
	"Africa/Tunis": "TN", "Africa/Tripoli": "LY", "Africa/Khartoum": "SD",
	"Africa/Lagos": "NG", "Africa/Nairobi": "KE", "Africa/Mogadishu": "SO",
	"Africa/Johannesburg": "ZA", "Africa/Addis_Ababa": "ET",
	"Asia/Riyadh": "SA", "Asia/Dubai": "AE", "Asia/Qatar": "QA",
	"Asia/Kuwait": "KW", "Asia/Bahrain": "BH", "Asia/Muscat": "OM",
	"Asia/Amman": "JO", "Asia/Beirut": "LB", "Asia/Baghdad": "IQ",
	"Asia/Damascus": "SY", "Asia/Jerusalem": "IL", "Asia/Gaza": "PS",
	"Asia/Tehran": "IR", "Asia/Karachi": "PK", "Asia/Kolkata": "IN",
	"Asia/Dhaka": "BD", "Asia/Kabul": "AF", "Asia/Colombo": "LK",
	"Asia/Tashkent": "UZ", "Asia/Samarkand": "UZ", "Asia/Almaty": "KZ",
	"Asia/Bishkek": "KG", "Asia/Dushanbe": "TJ", "Asia/Ashgabat": "TM",
	"Asia/Baku": "AZ", "Asia/Tbilisi": "GE", "Asia/Yerevan": "AM",
	"Asia/Jakarta": "ID", "Asia/Kuala_Lumpur": "MY", "Asia/Brunei": "BN",
	"Asia/Singapore": "SG", "Asia/Bangkok": "TH", "Asia/Manila": "PH",
	"Asia/Shanghai": "CN", "Asia/Tokyo": "JP", "Asia/Seoul": "KR",
	"Europe/Istanbul": "TR", "Europe/London": "GB", "Europe/Paris": "FR",
	"Europe/Berlin": "DE", "Europe/Madrid": "ES", "Europe/Rome": "IT",
	"Europe/Amsterdam": "NL", "Europe/Zurich": "CH", "Europe/Warsaw": "PL",
	"Europe/Moscow": "RU", "Europe/Kiev": "UA", "Europe/Kyiv": "UA",
	"Europe/Minsk":     "BY",
	"America/New_York": "US", "America/Chicago": "US", "America/Denver": "US",
	"America/Los_Angeles": "US", "America/Toronto": "CA", "America/Mexico_City": "MX",
	"America/Sao_Paulo": "BR", "Australia/Sydney": "AU",
}

// CurrencyForCountry returns the ISO 4217 currency for an ISO 3166-1 alpha-2
// country code, or "" if the country is unknown or empty.
func CurrencyForCountry(countryCode string) string {
	return currencyByCountry[strings.ToUpper(strings.TrimSpace(countryCode))]
}

// CurrencyForTimezone returns a best-effort currency derived from an IANA
// timezone, or "" if the timezone is not mapped.
func CurrencyForTimezone(timezone string) string {
	return CurrencyForCountry(timezoneCountry[strings.TrimSpace(timezone)])
}
