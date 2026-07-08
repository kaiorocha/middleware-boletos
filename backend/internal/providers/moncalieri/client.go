package moncalieri

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	providererrors "github.com/kaiorocha/middleware-boletos/backend/internal/providers/errors"
)

type client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func newClient(cfg Config) *client {
	timeout := 30 * time.Second
	if cfg.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
	}
	return &client{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:     cfg.APIKey,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *client) post(ctx context.Context, path string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return providererrors.New(errInvalidRequest, "failed to encode request payload", providerName, false)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return providererrors.New(errInvalidProviderConfig, "failed to build provider request", providerName, false)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return providererrors.New(errProviderHTTP, fmt.Sprintf("provider request failed: %v", err), providerName, true)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return providererrors.New(errProviderHTTP, "failed to read provider response", providerName, true)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		perr := providererrors.New(errProviderHTTP, fmt.Sprintf("provider returned HTTP %d", resp.StatusCode), providerName, resp.StatusCode >= 500)
		perr.HTTPStatus = resp.StatusCode
		return perr
	}
	if len(respBody) == 0 {
		return providererrors.New(errProviderUnexpected, "provider returned empty response", providerName, false)
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return providererrors.New(errProviderUnexpected, "failed to decode provider response", providerName, false)
	}
	return nil
}
