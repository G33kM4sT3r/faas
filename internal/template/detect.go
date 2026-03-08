// Package template provides language detection and template rendering.
package template

import (
	"fmt"
	"path/filepath"
	"strings"
)

var extToLang = map[string]string{ //nolint:gochecknoglobals // immutable lookup table
	".go":  "go",
	".py":  "python",
	".rs":  "rust",
	".php": "php",
	".ts":  "typescript",
	".js":  "javascript",
}

// SupportedLanguages returns a list of supported language names.
func SupportedLanguages() []string {
	seen := make(map[string]struct{}, len(extToLang))
	langs := make([]string, 0, len(extToLang))
	for _, lang := range extToLang {
		if _, ok := seen[lang]; !ok {
			seen[lang] = struct{}{}
			langs = append(langs, lang)
		}
	}
	return langs
}

// Detect returns the language name for a given filename based on extension.
func Detect(filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return "", fmt.Errorf("no file extension: %s", filename)
	}

	lang, ok := extToLang[ext]
	if !ok {
		return "", fmt.Errorf("unsupported language for extension %s (supported: %s)",
			ext, strings.Join(SupportedLanguages(), ", "))
	}

	return lang, nil
}
