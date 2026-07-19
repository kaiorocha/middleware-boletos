package service

import "errors"

var (
	ErrValidation        = errors.New("validation error")
	ErrNotFound          = errors.New("not found")
	ErrDuplicateResource = errors.New("duplicate resource")
	ErrCustomerBlocked   = errors.New("customer blocked")
)

type DuplicateResourceError struct {
	Message string
}

func NewDuplicateResource(message string) error {
	if message == "" {
		message = "Já existe um recurso com estes dados neste tenant."
	}
	return DuplicateResourceError{Message: message}
}

func (e DuplicateResourceError) Error() string {
	return e.Message
}

func (e DuplicateResourceError) Is(target error) bool {
	return target == ErrDuplicateResource
}

type CustomerBlockedError struct {
	Message string
}

func NewCustomerBlocked(message string) error {
	if message == "" {
		message = "Este cliente está bloqueado para novas emissões."
	}
	return CustomerBlockedError{Message: message}
}

func (e CustomerBlockedError) Error() string {
	return e.Message
}

func (e CustomerBlockedError) Is(target error) bool {
	return target == ErrCustomerBlocked
}
