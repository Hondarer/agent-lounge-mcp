package tools

import "testing"

func TestFinalToolNames(t *testing.T) {
	catalog := Catalog()
	for _, name := range []string{"lounge_join", "lounge_who", "chat_send", "chat_inbox", "lounge_info"} {
		if _, ok := catalog[name]; !ok {
			t.Errorf("missing tool %s", name)
		}
	}
}
