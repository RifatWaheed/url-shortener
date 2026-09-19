package main

import (
	"crypto/rand"
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const codeLen = 7

func GenerateShortCode() (string, error) {
	code := make([]byte, 0, codeLen)
	buf := make([]byte, codeLen*2)

	for len(code) < codeLen {
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}

		for _, b := range buf {
			if b < 248 {
				code = append(code, alphabet[b%62])
				if len(code) == codeLen {
					break
				}
			}
		}

	}

	return string(code), nil
}
