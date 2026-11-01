package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testBaseURL = "http://test.local"

type fakeStore struct {
	code    string
	longURL string
	err     error
}

func (f *fakeStore) CreateLink(ctx context.Context, longURL string) (string, error) {
	return f.code, f.err
}
func (f *fakeStore) GetLongUrl(ctx context.Context, shortCode string) (string, error) {
	return f.longURL, f.err
}

var _ LinkStore = (*fakeStore)(nil)

type handlerCase struct {
	method       string
	path         string
	contentType  string
	body         string
	store        *fakeStore
	wantStatus   int
	wantCode     string // "" = don't check the JSON envelope
	wantShortURL string // "" = don't check the shorten response
	wantLocation string // "" = don't check the redirect target
}

func TestHandlers(t *testing.T) {
	validBody := `{"url":"https://example.com"}`

	tests := map[string]handlerCase{
		// POST /shorten
		"POST /shorten wrong content type -> 415": {
			method:      http.MethodPost,
			path:        "/shorten",
			contentType: "application/xml",
			body:        validBody,
			wantStatus:  http.StatusUnsupportedMediaType,
			wantCode:    "content_type_invalid",
		},
		"POST /shorten body over 1MB -> 413": {
			method:      http.MethodPost,
			path:        "/shorten",
			contentType: "application/json",
			body:        `{"url":"https://example.com/` + strings.Repeat("a", 1<<20) + `"}`,
			wantStatus:  http.StatusRequestEntityTooLarge,
			wantCode:    "body_too_large",
		},
		"POST /shorten malformed JSON -> 400": {
			method:      http.MethodPost,
			path:        "/shorten",
			contentType: "application/json",
			body:        `{"url":`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    "json_invalid",
		},
		"POST /shorten empty url -> 400": {
			method:      http.MethodPost,
			path:        "/shorten",
			contentType: "application/json",
			body:        `{"url":""}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    "url_required",
		},
		"POST /shorten ftp scheme -> 400": {
			method:      http.MethodPost,
			path:        "/shorten",
			contentType: "application/json",
			body:        `{"url":"ftp://example.com"}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    "url_scheme_invalid",
		},
		"POST /shorten success -> 201": {
			method:       http.MethodPost,
			path:         "/shorten",
			contentType:  "application/json",
			body:         validBody,
			store:        &fakeStore{code: "abc1234"},
			wantStatus:   http.StatusCreated,
			wantShortURL: testBaseURL + "/abc1234",
		},
		"POST /shorten code exhausted -> 503": {
			method:      http.MethodPost,
			path:        "/shorten",
			contentType: "application/json",
			body:        validBody,
			store:       &fakeStore{err: ErrCodeExhausted},
			wantStatus:  http.StatusServiceUnavailable,
			wantCode:    "short_code_exhausted",
		},
		"POST /shorten store failure -> 500": {
			method:      http.MethodPost,
			path:        "/shorten",
			contentType: "application/json",
			body:        validBody,
			store:       &fakeStore{err: errors.New("connection refused")},
			wantStatus:  http.StatusInternalServerError,
			wantCode:    "internal_server_error",
		},

		// GET /{code}
		"GET malformed code -> 404": {
			method:     http.MethodGet,
			path:       "/abc-123",
			wantStatus: http.StatusNotFound,
			wantCode:   "link_not_found",
		},
		"GET unknown code -> 404": {
			method:     http.MethodGet,
			path:       "/abc1234",
			store:      &fakeStore{err: ErrLinkNotFound},
			wantStatus: http.StatusNotFound,
			wantCode:   "link_not_found",
		},
		"GET expired code -> 410": {
			method:     http.MethodGet,
			path:       "/abc1234",
			store:      &fakeStore{err: ErrLinkExpired},
			wantStatus: http.StatusGone,
			wantCode:   "link_expired",
		},
		"GET valid code -> 302": {
			method:       http.MethodGet,
			path:         "/abc1234",
			store:        &fakeStore{longURL: "https://example.com/page"},
			wantStatus:   http.StatusFound,
			wantLocation: "https://example.com/page",
		},

		// method routing
		"DELETE /shorten -> 405": {
			method:     http.MethodDelete,
			path:       "/shorten",
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.store == nil {
				tc.store = &fakeStore{}
			}
			srv := &Server{db: tc.store, baseURL: testBaseURL}

			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}

			rec := httptest.NewRecorder()
			srv.routes().ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tc.wantStatus, rec.Body.String())
			}

			if tc.wantCode != "" {
				assertErrorCode(t, rec, tc.wantCode)
			}

			if tc.wantShortURL != "" {
				var resp ShortenResponse
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("decode shorten response: %v", err)
				}
				if resp.ShortURL != tc.wantShortURL {
					t.Errorf("shortUrl = %q, want %q", resp.ShortURL, tc.wantShortURL)
				}
			}

			if tc.wantLocation != "" {
				if got := rec.Header().Get("Location"); got != tc.wantLocation {
					t.Errorf("Location = %q, want %q", got, tc.wantLocation)
				}
			}
		})
	}
}

// assertErrorCode decodes the JSON error envelope and checks its code field.
func assertErrorCode(t *testing.T, rec *httptest.ResponseRecorder, wantCode string) {
	t.Helper()

	var resp errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error envelope: %v; body: %s", err, rec.Body.String())
	}
	if resp.Error.Code != wantCode {
		t.Errorf("error code = %q, want %q", resp.Error.Code, wantCode)
	}
}
