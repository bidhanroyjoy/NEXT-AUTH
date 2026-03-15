package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
)

// GenerateRandomBytes returns securely generated random bytes.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// GenerateRandomString returns a URL-safe, base64 encoded securely generated random string.
func GenerateRandomString(s int) (string, error) {
	b, err := GenerateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b), err
}

// CSRFMiddleware validates CSRF tokens for state-mutating requests
func CSRFMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// For safe methods, generate a token if not present
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			cookie, _ := r.Cookie("csrf_token")
			if cookie == nil {
				newToken, _ := GenerateRandomString(32)
				http.SetCookie(w, &http.Cookie{
					Name:     "csrf_token",
					Value:    newToken,
					Path:     "/",
					MaxAge:   3600,
					HttpOnly: false,
				})
			}
			next(w, r)
			return
		}

		// Validate CSRF token for state-mutating methods
		csrfCookie, err := r.Cookie("csrf_token")
		csrfHeader := r.Header.Get("X-CSRF-Token")

		if err != nil || csrfHeader == "" || csrfCookie.Value != csrfHeader {
			writeError(w, http.StatusForbidden, "CSRF token mismatch or missing")
			return
		}

		next(w, r)
	}
}

// formatUint converts a uint to string for passing in headers
func formatUint(n uint) string {
	return fmt.Sprintf("%d", n)
}

// parseUint parses a string to uint
func parseUint(s string) uint {
	var n uint
	fmt.Sscanf(s, "%d", &n)
	return n
}

// getCookie retrieves a cookie value by name
func getCookie(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}
