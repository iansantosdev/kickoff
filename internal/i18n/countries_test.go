package i18n

import (
	"testing"
)

func TestNormalizeCountry_Alpha2(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"BR", "BR"},
		{"US", "US"},
		{"GB", "GB"},
		{"br", "BR"},
		{"  br  ", "BR"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := NormalizeCountry(tt.input); got != tt.want {
				t.Errorf("NormalizeCountry(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeCountry_FIFACodes(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"BRA", "BR"},
		{"USA", "US"},
		{"ARG", "AR"},
		{"GER", "DE"},
		{"FRA", "FR"},
		{"ESP", "ES"},
		{"ITA", "IT"},
		{"POR", "PT"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := NormalizeCountry(tt.input); got != tt.want {
				t.Errorf("NormalizeCountry(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeCountry_EnglishNames(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Brazil", "BR"},
		{"BRAZIL", "BR"},
		{"United States", "US"},
		{"Germany", "DE"},
		{"France", "FR"},
		{"United Kingdom", "GB"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := NormalizeCountry(tt.input); got != tt.want {
				t.Errorf("NormalizeCountry(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeCountry_PortugueseNames(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Brasil", "BR"},
		{"Estados Unidos", "US"},
		{"Alemanha", "DE"},
		{"França", "FR"},
		{"Reino Unido", "GB"},
		{"África do Sul", "ZA"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := NormalizeCountry(tt.input); got != tt.want {
				t.Errorf("NormalizeCountry(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeCountry_Empty(t *testing.T) {
	if got := NormalizeCountry(""); got != "" {
		t.Errorf("NormalizeCountry(\"\") = %q, want empty", got)
	}

	if got := NormalizeCountry("  "); got != "" {
		t.Errorf("NormalizeCountry(spaces) = %q, want empty", got)
	}

	if got := NormalizeCountry("unknown-land"); got != "UNKNOWN-LAND" {
		t.Errorf("NormalizeCountry(unknown) = %q, want %q", got, "UNKNOWN-LAND")
	}
}

func TestNormalizeCountry_InformalCode(t *testing.T) {
	if got := NormalizeCountry("UK"); got != "GB" {
		t.Errorf("NormalizeCountry(\"UK\") = %q, want %q", got, "GB")
	}
}

func TestCountryFromLang(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"pt-BR", "BR"},
		{"pt_br", "BR"},
		{"en-US", "US"},
		{"en_US", "US"},
		{"pt", "BR"},
		{"en", ""},
		{"fr", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := CountryFromLang(tt.input); got != tt.want {
				t.Errorf("CountryFromLang(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
