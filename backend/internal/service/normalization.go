package service

import (
	"strings"
	"unicode"
)

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func NormalizeDocument(document *string) *string {
	if document == nil {
		return nil
	}
	var b strings.Builder
	for _, r := range *document {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	v := b.String()
	if v == "" {
		return nil
	}
	return &v
}

func NormalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	v := strings.TrimSpace(*value)
	if v == "" {
		return nil
	}
	return &v
}

func NormalizeOptionalEmail(value *string) *string {
	if value == nil {
		return nil
	}
	v := NormalizeEmail(*value)
	if v == "" {
		return nil
	}
	return &v
}

func NormalizePostalCode(value *string) *string {
	return NormalizeDocument(value)
}
