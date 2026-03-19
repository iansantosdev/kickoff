// Package i18n provides internationalization support.
package i18n

import (
	"strings"
	"sync"
)

// Bundle stores an immutable language selection for lookups.
type Bundle struct {
	language string
}

// New creates a translation bundle for the provided language code.
func New(lang string) Bundle {
	return Bundle{language: normalizeLanguage(lang)}
}

// IsZero reports whether the bundle has not been initialized explicitly.
func (b Bundle) IsZero() bool {
	return b.language == ""
}

// Language returns the normalized language code for the bundle.
func (b Bundle) Language() string {
	if b.language == "" {
		return "en"
	}
	return b.language
}

// Get returns the translated string for the given key.
// Falls back to the key itself if no translation is found.
func (b Bundle) Get(key string) string {
	dict := dictEN
	if b.Language() == "pt-BR" {
		dict = dictPTBR
	}

	if val, ok := dict[key]; ok {
		return val
	}
	return key
}

// Keys returns all translation keys for the bundle language.
func (b Bundle) Keys() []string {
	dict := dictEN
	if b.Language() == "pt-BR" {
		dict = dictPTBR
	}
	keys := make([]string, 0, len(dict))
	for k := range dict {
		keys = append(keys, k)
	}
	return keys
}

var (
	mu            sync.RWMutex
	defaultBundle = New("en")
)

// SetLanguage configures the active language for translations.
// Supported values: "en", "pt", "pt-BR", "pt_br".
func SetLanguage(lang string) {
	mu.Lock()
	defer mu.Unlock()
	defaultBundle = New(lang)
}

// CurrentLanguage returns the active language code.
func CurrentLanguage() string {
	mu.RLock()
	defer mu.RUnlock()
	return defaultBundle.Language()
}

// Default returns the process-wide translation bundle.
func Default() Bundle {
	mu.RLock()
	defer mu.RUnlock()
	return defaultBundle
}

// Get returns the translated string for the given key.
// Falls back to the key itself if no translation is found.
func Get(key string) string {
	return Default().Get(key)
}

// Keys returns all translation keys for the given language.
// Useful for testing consistency between dictionaries.
func Keys(lang string) []string {
	return New(lang).Keys()
}

func normalizeLanguage(lang string) string {
	normalized := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(lang, "_", "-")))

	switch normalized {
	case "pt", "pt-br":
		return "pt-BR"
	default:
		return "en"
	}
}
