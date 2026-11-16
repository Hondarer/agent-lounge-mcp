package config

import "testing"

func TestValidate(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatalf("default configuration: %v", err)
	}
	for _, listen := range []string{"127.0.0.1:0", "[::1]:7788", "localhost:7788"} {
		cfg := Default()
		cfg.Listen = listen
		if err := cfg.Validate(); err != nil {
			t.Errorf("--listen %s: %v", listen, err)
		}
	}
	for _, listen := range []string{"0.0.0.0:7788", ":7788", "192.0.2.1:7788", "example.com:7788", "127.0.0.1"} {
		cfg := Default()
		cfg.Listen = listen
		if err := cfg.Validate(); err == nil {
			t.Errorf("--listen %s was accepted", listen)
		}
	}
	for _, db := range []string{"", "lounge.db?mode=memory"} {
		cfg := Default()
		cfg.DBPath = db
		if err := cfg.Validate(); err == nil {
			t.Errorf("--db %q was accepted", db)
		}
	}
}
