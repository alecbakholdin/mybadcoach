package main

import "testing"

func TestInitialViewLine(t *testing.T) {
	const want = "Initial view: LINEUP_SETUP"

	if got := initialViewLine(); got != want {
		t.Fatalf("initialViewLine() = %q, want %q", got, want)
	}
}
