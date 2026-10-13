package main

import (
	"context"
	"testing"
)

type fakeStore struct {
	code    string
	longURL string
	err     error
}

func (f *fakeStore) CreateLink(ctx context.Context, longURL string) (string, error)
func (f *fakeStore) GetLongUrl(ctx context.Context, shortCode string) (string, error)

var _ LinkStore = (*fakeStore)(nil)

type handlerCase struct {
	method      string
	path        string
	contentType string
	body        string
	store       *fakeStore
	wantStatus  int
	wantCode    string // "" = don't check the JSON envelope
}

func TestHandlers(t *testing.t) {
	tests := map[string]handlerCase{}

}
