package slugify

import "testing"

func TestSlug(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"My Bad, Coach", "my-bad-coach"},
		{"  Leading & Trailing!  ", "leading-trailing"},
		{"José García", "jose-garcia"},
		{"O'Brien", "o-brien"},
		{"already--double---hyphens", "already-double-hyphens"},
		{"UPPER case", "upper-case"},
		{"Bob 日本語 Smith", "bob-smith"},
		{"日本語", ""},
		{"", ""},
		{"!!!", ""},
	}
	for _, tt := range tests {
		if got := Slug(tt.in); got != tt.want {
			t.Errorf("Slug(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
