package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestRequestID_GeneratesWhenAbsent(t *testing.T) {
	app := &App{}
	h := app.requestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v, ok := r.Context().Value(requestIDKey).(string); ok {
			w.Header().Set("X-Context-Request-ID", v)
		}
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	rid := rr.Header().Get("X-Request-ID")
	if rid == "" {
		t.Fatalf("expected X-Request-ID header")
	}
	if _, err := uuid.Parse(rid); err != nil {
		t.Fatalf("expected valid uuid, got %v", rid)
	}
	ctxID := rr.Header().Get("X-Context-Request-ID")
	if ctxID != rid {
		t.Fatalf("context id and header differ: %s vs %s", ctxID, rid)
	}
}

func TestRequestID_PreservesValidUUID(t *testing.T) {
	app := &App{}
	valid := uuid.New().String()
	h := app.requestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", valid)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Header().Get("X-Request-ID") != valid {
		t.Fatalf("expected preserved request id")
	}
}

func TestRequestID_ReplacesInvalidOrTooLong(t *testing.T) {
	app := &App{}
	invalid := "not-a-uuid"
	h := app.requestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v, ok := r.Context().Value(requestIDKey).(string); ok {
			w.Header().Set("X-Context-Request-ID", v)
		}
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", invalid)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	rid := rr.Header().Get("X-Request-ID")
	if rid == invalid {
		t.Fatalf("expected different id for invalid header")
	}
	if _, err := uuid.Parse(rid); err != nil {
		t.Fatalf("expected generated uuid, got %v", rid)
	}

	long := strings.Repeat("a", 129)
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("X-Request-ID", long)
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Header().Get("X-Request-ID") == long {
		t.Fatalf("expected long header to be replaced")
	}
}
