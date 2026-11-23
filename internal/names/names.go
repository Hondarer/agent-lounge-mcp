// Package names validates and normalizes lounge identifiers.
package names

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/secure/precis"
)

const characters = "1 to 64 normalized Unicode letters, digits, combining marks, or ASCII . _ - @"

const (
	SessionRule = "Session names use " + characters + " and preserve case."
	UserRule    = "User names use " + characters + " and are lowercased."
)

// Session uses RFC 8265 to fold width variants, normalize to NFC, and reject
// disallowed invisible characters and invalid bidirectional text.
func Session(name string) (string, error) {
	return normalize(name, precis.UsernameCasePreserved, "name", SessionRule)
}

// User applies the same RFC 8265 rules as Session and maps case so accounts
// cannot differ only in case.
func User(name string) (string, error) {
	return normalize(name, precis.UsernameCaseMapped, "user name", UserRule)
}

func normalize(name string, profile *precis.Profile, label, rule string) (string, error) {
	normalized, err := profile.String(name)
	if err != nil {
		return "", fmt.Errorf("invalid %s %q: %s", label, name, rule)
	}
	for _, r := range normalized {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsMark(r) && !strings.ContainsRune("._-@", r) {
			return "", fmt.Errorf("invalid %s %q: %s", label, name, rule)
		}
	}
	if count := utf8.RuneCountInString(normalized); count < 1 || count > 64 {
		return "", fmt.Errorf("invalid %s %q: %s", label, name, rule)
	}
	return normalized, nil
}
