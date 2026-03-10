package i18n

import "strings"

// NormalizeCountry maps common country identifiers to ISO 3166-1 alpha-2 codes.
// Accepts: ISO alpha-2 (BR), FIFA/IOC codes (BRA), or country names (Brazil, Brasil).
func NormalizeCountry(input string) string {
	if input == "" {
		return ""
	}
	upper := strings.ToUpper(strings.TrimSpace(input))
	if code, ok := countryAliases[upper]; ok {
		return code
	}
	// If it's already 2 letters, assume it's a valid alpha-2 code
	if len(upper) == 2 {
		return upper
	}
	return upper
}

var countryAliases = map[string]string{
	// FIFA/IOC 3-letter codes → ISO alpha-2
	"AFG": "AF", // Afghanistan
	"ALB": "AL", // Albania
	"ALG": "DZ", // Algeria
	"AND": "AD", // Andorra
	"ANG": "AO", // Angola
	"ARG": "AR", // Argentina
	"ARM": "AM", // Armenia
	"AUS": "AU", // Australia
	"AUT": "AT", // Austria
	"AZE": "AZ", // Azerbaijan
	"BAN": "BD", // Bangladesh
	"BEL": "BE", // Belgium
	"BIH": "BA", // Bosnia and Herzegovina
	"BLR": "BY", // Belarus
	"BOL": "BO", // Bolivia
	"BRA": "BR", // Brazil
	"BUL": "BG", // Bulgaria
	"CAN": "CA", // Canada
	"CHI": "CL", // Chile
	"CHN": "CN", // China
	"COL": "CO", // Colombia
	"CRC": "CR", // Costa Rica
	"CRO": "HR", // Croatia
	"CUB": "CU", // Cuba
	"CYP": "CY", // Cyprus
	"CZE": "CZ", // Czech Republic
	"DEN": "DK", // Denmark
	"ECU": "EC", // Ecuador
	"EGY": "EG", // Egypt
	"ENG": "GB", // England
	"ESP": "ES", // Spain (Español)
	"EST": "EE", // Estonia
	"ETH": "ET", // Ethiopia
	"FIN": "FI", // Finland
	"FRA": "FR", // France
	"GEO": "GE", // Georgia
	"GER": "DE", // Germany
	"GHA": "GH", // Ghana
	"GRE": "GR", // Greece
	"GUA": "GT", // Guatemala
	"HOL": "NL", // Holland
	"HON": "HN", // Honduras
	"HUN": "HU", // Hungary
	"ICE": "IS", // Iceland
	"INA": "ID", // Indonesia
	"IND": "IN", // India
	"IRI": "IR", // Iran
	"IRL": "IE", // Ireland
	"IRQ": "IQ", // Iraq
	"ISR": "IL", // Israel
	"ITA": "IT", // Italy
	"JAM": "JM", // Jamaica
	"JAP": "JP", // Japan
	"JPN": "JP", // Japan
	"KAZ": "KZ", // Kazakhstan
	"KEN": "KE", // Kenya
	"KOR": "KR", // South Korea
	"KSA": "SA", // Saudi Arabia
	"KUW": "KW", // Kuwait
	"LAT": "LV", // Latvia
	"LIB": "LB", // Lebanon
	"LTU": "LT", // Lithuania
	"LUX": "LU", // Luxembourg
	"MAR": "MA", // Morocco
	"MAS": "MY", // Malaysia
	"MEX": "MX", // Mexico
	"MNE": "ME", // Montenegro
	"MON": "MC", // Monaco
	"NED": "NL", // Netherlands
	"NGR": "NG", // Nigeria
	"NIR": "GB", // Northern Ireland
	"NOR": "NO", // Norway
	"NZL": "NZ", // New Zealand
	"OMA": "OM", // Oman
	"PAK": "PK", // Pakistan
	"PAN": "PA", // Panama
	"PAR": "PY", // Paraguay
	"PER": "PE", // Peru
	"PHI": "PH", // Philippines
	"POL": "PL", // Poland
	"POR": "PT", // Portugal
	"QAT": "QA", // Qatar
	"ROU": "RO", // Romania
	"RSA": "ZA", // South Africa
	"RUS": "RU", // Russia
	"SCO": "GB", // Scotland
	"SEN": "SN", // Senegal
	"SIN": "SG", // Singapore
	"SLO": "SI", // Slovenia
	"SPA": "ES", // Spain
	"SRB": "RS", // Serbia
	"SUI": "CH", // Switzerland
	"SVK": "SK", // Slovakia
	"SWE": "SE", // Sweden
	"THA": "TH", // Thailand
	"TUN": "TN", // Tunisia
	"TUR": "TR", // Turkey
	"UAE": "AE", // United Arab Emirates
	"UKR": "UA", // Ukraine
	"URU": "UY", // Uruguay
	"USA": "US", // United States
	"UZB": "UZ", // Uzbekistan
	"VEN": "VE", // Venezuela
	"VIE": "VN", // Vietnam
	"WAL": "GB", // Wales

	// Common informal codes
	"UK": "GB", // United Kingdom

	// Country names (English)
	"AFGHANISTAN":            "AF",
	"ALBANIA":                "AL",
	"ALGERIA":                "DZ",
	"ANDORRA":                "AD",
	"ANGOLA":                 "AO",
	"ARGENTINA":              "AR",
	"ARMENIA":                "AM",
	"AUSTRALIA":              "AU",
	"AUSTRIA":                "AT",
	"AZERBAIJAN":             "AZ",
	"BANGLADESH":             "BD",
	"BELGIUM":                "BE",
	"BOLIVIA":                "BO",
	"BOSNIA AND HERZEGOVINA": "BA",
	"BRAZIL":                 "BR",
	"BULGARIA":               "BG",
	"CANADA":                 "CA",
	"CHILE":                  "CL",
	"CHINA":                  "CN",
	"COLOMBIA":               "CO",
	"COSTA RICA":             "CR",
	"CROATIA":                "HR",
	"CUBA":                   "CU",
	"CYPRUS":                 "CY",
	"CZECH REPUBLIC":         "CZ",
	"CZECHIA":                "CZ",
	"DENMARK":                "DK",
	"ECUADOR":                "EC",
	"EGYPT":                  "EG",
	"ENGLAND":                "GB",
	"ESTONIA":                "EE",
	"ETHIOPIA":               "ET",
	"FINLAND":                "FI",
	"FRANCE":                 "FR",
	"GEORGIA":                "GE",
	"GERMANY":                "DE",
	"GHANA":                  "GH",
	"GREECE":                 "GR",
	"GUATEMALA":              "GT",
	"HONDURAS":               "HN",
	"HUNGARY":                "HU",
	"ICELAND":                "IS",
	"INDIA":                  "IN",
	"INDONESIA":              "ID",
	"IRAN":                   "IR",
	"IRAQ":                   "IQ",
	"IRELAND":                "IE",
	"ISRAEL":                 "IL",
	"ITALY":                  "IT",
	"JAMAICA":                "JM",
	"JAPAN":                  "JP",
	"KAZAKHSTAN":             "KZ",
	"KENYA":                  "KE",
	"KUWAIT":                 "KW",
	"LATVIA":                 "LV",
	"LEBANON":                "LB",
	"LITHUANIA":              "LT",
	"LUXEMBOURG":             "LU",
	"MALAYSIA":               "MY",
	"MEXICO":                 "MX",
	"MONACO":                 "MC",
	"MONTENEGRO":             "ME",
	"MOROCCO":                "MA",
	"NETHERLANDS":            "NL",
	"NEW ZEALAND":            "NZ",
	"NIGERIA":                "NG",
	"NORTH KOREA":            "KP",
	"NORTHERN IRELAND":       "GB",
	"NORWAY":                 "NO",
	"OMAN":                   "OM",
	"PAKISTAN":               "PK",
	"PANAMA":                 "PA",
	"PARAGUAY":               "PY",
	"PERU":                   "PE",
	"PHILIPPINES":            "PH",
	"POLAND":                 "PL",
	"PORTUGAL":               "PT",
	"QATAR":                  "QA",
	"ROMANIA":                "RO",
	"RUSSIA":                 "RU",
	"SAUDI ARABIA":           "SA",
	"SCOTLAND":               "GB",
	"SENEGAL":                "SN",
	"SERBIA":                 "RS",
	"SINGAPORE":              "SG",
	"SLOVAKIA":               "SK",
	"SLOVENIA":               "SI",
	"SOUTH AFRICA":           "ZA",
	"SOUTH KOREA":            "KR",
	"SPAIN":                  "ES",
	"SWEDEN":                 "SE",
	"SWITZERLAND":            "CH",
	"THAILAND":               "TH",
	"TUNISIA":                "TN",
	"TURKEY":                 "TR",
	"UKRAINE":                "UA",
	"UNITED ARAB EMIRATES":   "AE",
	"UNITED KINGDOM":         "GB",
	"UNITED STATES":          "US",
	"URUGUAY":                "UY",
	"UZBEKISTAN":             "UZ",
	"VENEZUELA":              "VE",
	"VIETNAM":                "VN",
	"WALES":                  "GB",

	// Country names (Portuguese)
	"AFEGANISTÃO":            "AF",
	"ALBÂNIA":                "AL",
	"ARGÉLIA":                "DZ",
	"ALEMANHA":               "DE",
	"ARÁBIA SAUDITA":         "SA",
	"ARMÊNIA":                "AM",
	"AUSTRÁLIA":              "AU",
	"ÁUSTRIA":                "AT",
	"AZERBAIJÃO":             "AZ",
	"BÉLGICA":                "BE",
	"BOLÍVIA":                "BO",
	"BÓSNIA E HERZEGOVINA":   "BA",
	"BRASIL":                 "BR",
	"BULGÁRIA":               "BG",
	"CANADÁ":                 "CA",
	"CATAR":                  "QA",
	"CAZAQUISTÃO":            "KZ",
	"CHIPRE":                 "CY",
	"COLÔMBIA":               "CO",
	"COREIA DO NORTE":        "KP",
	"COREIA DO SUL":          "KR",
	"CROÁCIA":                "HR",
	"DINAMARCA":              "DK",
	"EGITO":                  "EG",
	"EMIRADOS ÁRABES UNIDOS": "AE",
	"EQUADOR":                "EC",
	"ESCÓCIA":                "GB",
	"ESLOVÁQUIA":             "SK",
	"ESLOVÊNIA":              "SI",
	"ESPANHA":                "ES",
	"ESTADOS UNIDOS":         "US",
	"ESTÔNIA":                "EE",
	"ETIÓPIA":                "ET",
	"FILIPINAS":              "PH",
	"FINLÂNDIA":              "FI",
	"FRANÇA":                 "FR",
	"GANA":                   "GH",
	"GEÓRGIA":                "GE",
	"GRÉCIA":                 "GR",
	"HOLANDA":                "NL",
	"HUNGRIA":                "HU",
	"IÊMEN":                  "YE",
	"ÍNDIA":                  "IN",
	"INDONÉSIA":              "ID",
	"INGLATERRA":             "GB",
	"IRÃ":                    "IR",
	"IRAQUE":                 "IQ",
	"IRLANDA":                "IE",
	"IRLANDA DO NORTE":       "GB",
	"ISLÂNDIA":               "IS",
	"ITÁLIA":                 "IT",
	"JAPÃO":                  "JP",
	"LETÔNIA":                "LV",
	"LÍBANO":                 "LB",
	"LITUÂNIA":               "LT",
	"LUXEMBURGO":             "LU",
	"MALÁSIA":                "MY",
	"MARROCOS":               "MA",
	"MÉXICO":                 "MX",
	"MÔNACO":                 "MC",
	"NIGÉRIA":                "NG",
	"NORUEGA":                "NO",
	"NOVA ZELÂNDIA":          "NZ",
	"PAÍS DE GALES":          "GB",
	"PAÍSES BAIXOS":          "NL",
	"PAQUISTÃO":              "PK",
	"PARAGUAI":               "PY",
	"POLÔNIA":                "PL",
	"QUÊNIA":                 "KE",
	"REINO UNIDO":            "GB",
	"REPÚBLICA TCHECA":       "CZ",
	"ROMÊNIA":                "RO",
	"RÚSSIA":                 "RU",
	"SÉRVIA":                 "RS",
	"SUÉCIA":                 "SE",
	"SUÍÇA":                  "CH",
	"TAILÂNDIA":              "TH",
	"TUNÍSIA":                "TN",
	"TURQUIA":                "TR",
	"UCRÂNIA":                "UA",
	"URUGUAI":                "UY",
	"UZBEQUISTÃO":            "UZ",
	"ÁFRICA DO SUL":          "ZA",
}

// CountryFromLang extracts a country code from a language tag.
// e.g. "pt-BR" → "BR", "pt" → "BR", "en" → ""
func CountryFromLang(lang string) string {
	if parts := strings.SplitN(lang, "-", 2); len(parts) == 2 {
		return strings.ToUpper(parts[1])
	}
	// Map short language codes to default countries
	switch strings.ToLower(lang) {
	case "pt":
		return "BR"
	default:
		return ""
	}
}
