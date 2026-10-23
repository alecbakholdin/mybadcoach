package main

import "testing"

func TestEnvOrDefault(t *testing.T) {
	t.Setenv("MBC_TEST_SETTING", "value")
	if got := envOrDefault("MBC_TEST_SETTING", "fallback"); got != "value" {
		t.Fatalf("envOrDefault with a set variable = %q, want %q", got, "value")
	}

	t.Setenv("MBC_TEST_SETTING", "")
	if got := envOrDefault("MBC_TEST_SETTING", "fallback"); got != "fallback" {
		t.Fatalf("envOrDefault with an empty variable = %q, want %q", got, "fallback")
	}
}
