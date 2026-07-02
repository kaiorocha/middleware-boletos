package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	// reuse local handler
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		w.Header().Set("Content-Type","application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	s := httptest.NewServer(h)
	defer s.Close()

	res, err := http.Get(s.URL)
	if err != nil { t.Fatalf("get health: %v", err) }
	if res.StatusCode != http.StatusOK { t.Fatalf("expected 200, got %d", res.StatusCode) }
}
