package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecoverPanic(t *testing.T) {
	h := recoverPanic(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	assertErrorCode(t, rec, "internal_server_error")
}

func TestStatusRecorderKeepsFirstStatus(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
		w.WriteHeader(http.StatusInternalServerError)
	})

	sr := &statusRecorder{ResponseWriter: httptest.NewRecorder(), status: http.StatusOK}
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	h.ServeHTTP(sr, req)

	if sr.status != http.StatusOK {
		t.Errorf("recorded status = %d, want %d", sr.status, http.StatusOK)
	}
}
