package errors

import "fmt"

type ProviderError struct {
	Code         string `json:"code"`
	Message      string `json:"message"`
	Provider     string `json:"provider,omitempty"`
	Retryable    bool   `json:"retryable"`
	HTTPStatus   int    `json:"http_status,omitempty"`
	ResponseBody string `json:"-"`
}

func (e *ProviderError) Error() string {
	if e.Provider == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s %s: %s", e.Provider, e.Code, e.Message)
}

func New(code, message, provider string, retryable bool) *ProviderError {
	return &ProviderError{Code: code, Message: message, Provider: provider, Retryable: retryable}
}
