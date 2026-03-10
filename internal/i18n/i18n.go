// Package i18n provides internationalization support.
package i18n

import (
	"strings"
	"sync"
)

var (
	mu              sync.RWMutex
	currentLanguage = "en"
)

// SetLanguage configures the active language for translations.
// Supported values: "en", "pt", "pt-BR", "pt_br".
func SetLanguage(lang string) {
	lang = strings.ToLower(strings.TrimSpace(lang))

	mu.Lock()
	defer mu.Unlock()

	if lang == "pt" || lang == "pt-br" || lang == "pt_br" {
		currentLanguage = "pt-BR"
	} else {
		currentLanguage = "en"
	}
}

// CurrentLanguage returns the active language code.
func CurrentLanguage() string {
	mu.RLock()
	defer mu.RUnlock()
	return currentLanguage
}

// Get returns the translated string for the given key.
// Falls back to the key itself if no translation is found.
func Get(key string) string {
	mu.RLock()
	lang := currentLanguage
	mu.RUnlock()

	dict := dictEN
	if lang == "pt-BR" {
		dict = dictPTBR
	}

	if val, ok := dict[key]; ok {
		return val
	}
	return key
}

// Keys returns all translation keys for the given language.
// Useful for testing consistency between dictionaries.
func Keys(lang string) []string {
	dict := dictEN
	if lang == "pt-BR" {
		dict = dictPTBR
	}
	keys := make([]string, 0, len(dict))
	for k := range dict {
		keys = append(keys, k)
	}
	return keys
}
