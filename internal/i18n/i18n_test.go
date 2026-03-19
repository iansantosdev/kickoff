package i18n

import (
	"testing"
)

func TestBundleZeroAndLanguage(t *testing.T) {
	var zero Bundle
	if !zero.IsZero() {
		t.Fatal("zero bundle should report IsZero")
	}
	if got := zero.Language(); got != "en" {
		t.Fatalf("zero bundle language = %q, want %q", got, "en")
	}

	pt := New("pt_br")
	if pt.IsZero() {
		t.Fatal("initialized bundle should not be zero")
	}
	if got := pt.Language(); got != "pt-BR" {
		t.Fatalf("Language() = %q, want %q", got, "pt-BR")
	}
}

func TestSetLanguage(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"en", "en"},
		{"pt", "pt-BR"},
		{"pt-BR", "pt-BR"},
		{"pt-br", "pt-BR"},
		{"pt_br", "pt-BR"},
		{"PT-BR", "pt-BR"},
		{"  pt-br  ", "pt-BR"},
		{"fr", "en"}, // unsupported falls back to en
		{"", "en"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			SetLanguage(tt.input)
			if got := CurrentLanguage(); got != tt.want {
				t.Errorf("SetLanguage(%q): CurrentLanguage() = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGet_EnglishKeys(t *testing.T) {
	SetLanguage("en")

	tests := []struct {
		key  string
		want string
	}{
		{"live", "LIVE"},
		{"full_time", "Full Time"},
		{"halftime", "Halftime"},
		{"round", "Round"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := Get(tt.key); got != tt.want {
				t.Errorf("Get(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestGet_PortugueseKeys(t *testing.T) {
	SetLanguage("pt-BR")

	tests := []struct {
		key  string
		want string
	}{
		{"live", "AO VIVO"},
		{"full_time", "Encerrado"},
		{"halftime", "Intervalo"},
		{"round", "Rodada"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := Get(tt.key); got != tt.want {
				t.Errorf("Get(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}

	// Reset to default
	SetLanguage("en")
}

func TestGet_MissingKeyFallsBackToKey(t *testing.T) {
	SetLanguage("en")

	key := "nonexistent_key_12345"
	if got := Get(key); got != key {
		t.Errorf("Get(%q) = %q, want key itself", key, got)
	}
}

func TestKeyConsistency(t *testing.T) {
	enKeys := make(map[string]bool)
	for _, k := range Keys("en") {
		enKeys[k] = true
	}

	ptKeys := make(map[string]bool)
	for _, k := range Keys("pt-BR") {
		ptKeys[k] = true
	}

	for k := range enKeys {
		if !ptKeys[k] {
			t.Errorf("key %q exists in EN but not in PT-BR", k)
		}
	}

	for k := range ptKeys {
		if !enKeys[k] {
			t.Errorf("key %q exists in PT-BR but not in EN", k)
		}
	}
}
