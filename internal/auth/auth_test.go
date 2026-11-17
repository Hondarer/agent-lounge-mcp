package auth

import (
	"net/http"
	"regexp"
	"testing"
)

func TestNewToken(t *testing.T) {
	form := regexp.MustCompile(`^alm_[A-Za-z0-9_-]{43}$`)
	seen := map[string]bool{}
	for range 100 {
		tok, err := NewToken()
		if err != nil {
			t.Fatal(err)
		}
		if !form.MatchString(tok) || len(tok) != 47 || !WellFormed(tok) {
			t.Fatalf("token %q has the wrong form", tok)
		}
		if seen[tok] {
			t.Fatal("a token was generated twice")
		}
		seen[tok] = true
	}
}

func TestHash(t *testing.T) {
	// SHA-256 of "abc".
	if got := Hash("abc"); got != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Errorf("Hash = %s", got)
	}
}

func TestWellFormed(t *testing.T) {
	tok, _ := NewToken()
	for _, s := range []string{"", "alm_", "xyz_" + tok[4:], tok + "A", tok[:46], "alm_" + "+" + tok[5:], "alm_" + "=" + tok[5:]} {
		if WellFormed(s) {
			t.Errorf("%q is taken as well formed", s)
		}
	}
}

func TestBearerToken(t *testing.T) {
	for _, tc := range []struct {
		values  []string
		token   string
		present bool
		bad     bool
	}{
		{nil, "", false, false},
		{[]string{""}, "", false, false},
		{[]string{"Bearer abc"}, "abc", true, false},
		{[]string{"bearer  abc "}, "abc", true, false},
		{[]string{"Basic abc"}, "", true, true},
		{[]string{"Bearer"}, "", true, true},
		{[]string{"Bearer "}, "", true, true},
		{[]string{"Bearer a b"}, "", true, true},
		{[]string{"abc"}, "", true, true},
		{[]string{"Bearer a", "Bearer b"}, "", true, true},
	} {
		h := http.Header{}
		for _, v := range tc.values {
			h.Add("Authorization", v)
		}
		token, present, err := BearerToken(h)
		if token != tc.token || present != tc.present || (err != nil) != tc.bad {
			t.Errorf("%q: got %q %v %v", tc.values, token, present, err)
		}
	}
}
