package names

import (
	"strings"
	"testing"
)

func TestNormalization(t *testing.T) {
	for _, tc := range []struct{ input, session, user string }{
		{"レビュー担当", "レビュー担当", "レビュー担当"},
		{"レビュー担当２", "レビュー担当2", "レビュー担当2"},
		{"ﾚﾋﾞｭｰ", "レビュー", "レビュー"},
		{"Ｒｅｖｉｅｗ", "Review", "review"},
		{"カ\u3099", "ガ", "ガ"},
		{"नमस्ते", "नमस्ते", "नमस्ते"},
		{"สวัสดี", "สวัสดี", "สวัสดี"},
		{"Reviewer", "Reviewer", "reviewer"},
		{"reviewer", "reviewer", "reviewer"},
		{"Alice", "Alice", "alice"},
		{"LOCAL", "LOCAL", "local"},
		{strings.Repeat("カ\u3099", 64), strings.Repeat("ガ", 64), strings.Repeat("ガ", 64)},
	} {
		t.Run(tc.input, func(t *testing.T) {
			if got, err := Session(tc.input); err != nil || got != tc.session {
				t.Errorf("Session = %q, %v, want %q", got, err, tc.session)
			}
			if got, err := User(tc.input); err != nil || got != tc.user {
				t.Errorf("User = %q, %v, want %q", got, err, tc.user)
			}
		})
	}
}

func TestInvalidNames(t *testing.T) {
	for _, input := range []string{
		"", "a b", "a😀", "a。", "①", "a\u200d", "a/b", "a:b", "a!", "a\n",
		"l·l", "क्\u200dष", "aא", strings.Repeat("ガ", 65),
	} {
		if _, err := Session(input); err == nil || !strings.Contains(err.Error(), SessionRule) {
			t.Errorf("Session(%q) error = %v", input, err)
		}
		if _, err := User(input); err == nil || !strings.Contains(err.Error(), UserRule) {
			t.Errorf("User(%q) error = %v", input, err)
		}
	}
}

func TestExistingNamesAreUnchanged(t *testing.T) {
	for _, name := range []string{"A", "0", "a.b_c-d@host", "._-@", strings.Repeat("Z", 64)} {
		if got, err := Session(name); err != nil || got != name {
			t.Errorf("Session(%q) = %q, %v", name, got, err)
		}
	}
	for _, name := range []string{"a", "0", "a.b_c-d", strings.Repeat("z", 32)} {
		if got, err := User(name); err != nil || got != name {
			t.Errorf("User(%q) = %q, %v", name, got, err)
		}
	}
}
