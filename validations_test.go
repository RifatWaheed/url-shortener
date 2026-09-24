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
		"exactlyAtLimit": {req: ShortenRequest{Url: validUrlHttps + strings.Repeat("a", 2048-len(validUrlHttps))}, wantErr: nil},
		"oneOverLimit":   {req: ShortenRequest{Url: validUrlHttps + strings.Repeat("a", 2049-len(validUrlHttps))}, wantErr: ErrUrlLengthCapExceeded},

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
