package main

import (
	"errors"
	"strings"
	"testing"
)

type testUrlStruct struct {
	req     ShortenRequest
	wantErr error
}
type testShortCode struct {
	shortCode string
	wantErr   error
}

func TestGenerateShortCode(t *testing.T) {
	hashSet := make(map[string]struct{})
	n := 10000

	for i := range n {
		code := GenerateShortCode()
		if len(code) != codeLen {
			t.Fatalf("got length %d, want %d (code %q)", len(code), codeLen, code)
		}
		if err := ValidateShortCode(code); err != nil {
			t.Fatalf("generated invalid code %q: %v", code, err)
		}
		if _, ok := hashSet[code]; ok {
			t.Fatalf("collision after %d codes: %q", i, code)
		}
		hashSet[code] = struct{}{}
	}

}

func TestValidateShortCode(t *testing.T) {
	tests := map[string]testShortCode{
		"emptyShortCode": {shortCode: "", wantErr: ErrShortCodeEmptyString},

		"shortCodeLengthExceeded": {shortCode: "a" + strings.Repeat("a", 400), wantErr: ErrUrlShortCodeLengthExceeded},
		"exactlyAtLimitShortCode": {shortCode: strings.Repeat("a", codeLen), wantErr: nil},
		"withinLimitShortCode":    {shortCode: strings.Repeat("a", codeLen-1), wantErr: nil},
		"oneOverLimitShortCode":   {shortCode: "a" + strings.Repeat("a", codeLen), wantErr: ErrUrlShortCodeLengthExceeded},

		"invalidCharHyphen":     {shortCode: "abc-123", wantErr: ErrShortCodeInvalidCharacter},
		"invalidCharUnderscore": {shortCode: "abc_123", wantErr: ErrShortCodeInvalidCharacter},
		"invalidCharSpace":      {shortCode: "abc 123", wantErr: ErrShortCodeInvalidCharacter},
		"invalidCharSlash":      {shortCode: "abc/12", wantErr: ErrShortCodeInvalidCharacter},
		"invalidCharAtStart":    {shortCode: "!abc12", wantErr: ErrShortCodeInvalidCharacter},
		"invalidCharAtEnd":      {shortCode: "abc12!", wantErr: ErrShortCodeInvalidCharacter},
		"invalidCharUnicode":    {shortCode: "abcé", wantErr: ErrShortCodeInvalidCharacter},
		"validMixedCase":        {shortCode: "aB3xY9z", wantErr: nil},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := ValidateShortCode(tc.shortCode)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got: %v, want: %v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateUrl(t *testing.T) {
	var validUrlHttps = "https://example.com/"
	var validUrlHttp = "http://example.com/"

	tests := map[string]testUrlStruct{
		"emptyUrl":          {req: ShortenRequest{Url: ""}, wantErr: ErrUrlEmpty},
		"lengthCapExceeded": {req: ShortenRequest{Url: validUrlHttps + strings.Repeat("a", 3000)}, wantErr: ErrUrlLengthCapExceeded},
		"validUrl":          {req: ShortenRequest{Url: validUrlHttps}, wantErr: nil},

		// Each rejection path
		"notAUrl":   {req: ShortenRequest{Url: "hello world"}, wantErr: ErrUrlParsingFailed},
		"ftpScheme": {req: ShortenRequest{Url: "ftp://example.com"}, wantErr: ErrUrlSchemeNotValid},
		"noHost":    {req: ShortenRequest{Url: "http://"}, wantErr: ErrUrlHostMissing},

		// Edges of the length limit
		"exactlyAtLimitUrl": {req: ShortenRequest{Url: validUrlHttps + strings.Repeat("a", 2048-len(validUrlHttps))}, wantErr: nil},
		"oneOverLimitUrl":   {req: ShortenRequest{Url: validUrlHttps + strings.Repeat("a", 2049-len(validUrlHttps))}, wantErr: ErrUrlLengthCapExceeded},

		// Other valid inputs
		"httpUrl":     {req: ShortenRequest{Url: validUrlHttp}, wantErr: nil},
		"urlWithPath": {req: ShortenRequest{Url: "https://example.com/a/b?x=1"}, wantErr: nil},
	}

	for name, tc := range tests {
		t.Run(name,
			func(t *testing.T) {
				err := ValidateUrl(tc.req)
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("got: %v, want: %v", err, tc.wantErr)
				}
			})
	}

}
