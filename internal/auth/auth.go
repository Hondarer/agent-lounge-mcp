// Package auth handles the tokens that identify users: generating them, hashing them for
// storage, and reading them from the Authorization header. It does string and random number
// processing only; the users and the hashes are kept by package store.
//
// Tokens are secrets. Nothing in this package logs a token or puts one in an error message.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
)

const (
	// TokenPrefix starts every token, so that a leaked token can be recognized as one of
	// agent-lounge-mcp (for example by secret scanners).
	TokenPrefix = "alm_"
	// tokenRandomBytes is the number of random bytes in a token.
	tokenRandomBytes = 32
)

// TokenLen is the length of a token: the prefix and 32 random bytes in unpadded URL-safe
// Base64 (43 characters).
var TokenLen = len(TokenPrefix) + base64.RawURLEncoding.EncodedLen(tokenRandomBytes)

// NewToken returns a new token made from the operating system's cryptographic random source.
func NewToken() (string, error) {
	var b [tokenRandomBytes]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return TokenPrefix + base64.RawURLEncoding.EncodeToString(b[:]), nil
}

// Hash returns the value stored for a token: its SHA-256 digest in lowercase hexadecimal.
// A token has 256 random bits, so a plain digest is enough; it needs no salt or slow hash.
func Hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// WellFormed reports whether s has the form of a token. A malformed value is rejected without
// looking it up.
func WellFormed(s string) bool {
	if len(s) != TokenLen || !strings.HasPrefix(s, TokenPrefix) {
		return false
	}
	_, err := base64.RawURLEncoding.Strict().DecodeString(s[len(TokenPrefix):])
	return err == nil
}

// ErrMalformedHeader is returned by BearerToken for an Authorization header that is not of the
// form "Bearer <token>".
var ErrMalformedHeader = errors.New("the Authorization header is not of the form Bearer <token>")

// BearerToken reads the token of the request's Authorization header. present is false when the
// request has no Authorization header (or an empty one). A header of another form, such as
// another scheme, fails with ErrMalformedHeader; the token is then "" and present is true.
// The scheme name is matched without regard to case (RFC 9110, section 11.1).
func BearerToken(h http.Header) (token string, present bool, err error) {
	values := h.Values("Authorization")
	if len(values) == 0 || (len(values) == 1 && strings.TrimSpace(values[0]) == "") {
		return "", false, nil
	}
	if len(values) > 1 {
		return "", true, ErrMalformedHeader
	}
	scheme, rest, ok := strings.Cut(strings.TrimSpace(values[0]), " ")
	token = strings.TrimSpace(rest)
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" || strings.ContainsAny(token, " \t") {
		return "", true, ErrMalformedHeader
	}
	return token, true, nil
}
