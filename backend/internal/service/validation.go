package service

import (
	"net/mail"
	"strings"

	"github.com/google/uuid"
)

func IsValidUUID(v string) bool {
	_, err := uuid.Parse(v)
	return err == nil
}

func IsValidEmail(v string) bool {
	if strings.TrimSpace(v) == "" {
		return false
	}
	_, err := mail.ParseAddress(v)
	return err == nil
}
