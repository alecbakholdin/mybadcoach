// Package slugify generates URL-safe slugs from display names.
//
// A slug is the lowercase, ASCII-safe, hyphen-separated form of a name, used
// as the stable URL identifier for teams, events, matches, and players. This
// package produces only the base slug from a name; uniqueness and collision
// resolution (appending "-2", "-3", …) is the caller's responsibility, since
// it requires a lookup against already-taken slugs.
package slugify

import (
	"strings"
	"unicode"
)

// Slug returns the slug form of name.
//
// It lowercases, strips common Latin diacritics, replaces each run of
// non-alphanumeric characters with a single hyphen, and trims leading and
// trailing hyphens. It returns "" when name contains no ASCII letter or
// digit (for example a name written entirely in a non-Latin script); callers
// should fall back to an ID in that case.
func Slug(name string) string {
	var b strings.Builder
	b.Grow(len(name))

	hyphen := false
	for _, r := range strings.ToLower(name) {
		r = stripDiacritic(r)
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			// strings.Builder writes never fail.
			_, _ = b.WriteRune(r)
			hyphen = false
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			// A letter we couldn't transliterate: drop it, but keep it from
			// colliding with an adjacent separator into a double hyphen.
			hyphen = true
		default:
			if !hyphen {
				// strings.Builder writes never fail.
				_ = b.WriteByte('-')
				hyphen = true
			}
		}
	}

	return strings.Trim(b.String(), "-")
}

// stripDiacritic maps a rune carrying a Latin diacritic to its ASCII base,
// returning r unchanged for everything else. It covers the common Western
// European cases; best-effort only, not a general transliterator.
func stripDiacritic(r rune) rune {
	switch r {
	case 'à', 'á', 'â', 'ã', 'ä', 'å', 'ā', 'ă', 'ą':
		return 'a'
	case 'ç', 'ć', 'č', 'ĉ', 'ċ':
		return 'c'
	case 'ď', 'đ':
		return 'd'
	case 'è', 'é', 'ê', 'ë', 'ē', 'ĕ', 'ė', 'ę', 'ě':
		return 'e'
	case 'ĝ', 'ğ', 'ġ', 'ģ':
		return 'g'
	case 'ĥ', 'ħ':
		return 'h'
	case 'ì', 'í', 'î', 'ï', 'ĩ', 'ī', 'ĭ', 'į', 'ı':
		return 'i'
	case 'ĵ':
		return 'j'
	case 'ķ':
		return 'k'
	case 'ĺ', 'ļ', 'ľ', 'ł':
		return 'l'
	case 'ñ', 'ń', 'ņ', 'ň':
		return 'n'
	case 'ò', 'ó', 'ô', 'õ', 'ö', 'ø', 'ō', 'ŏ', 'ő':
		return 'o'
	case 'ŕ', 'ŗ', 'ř':
		return 'r'
	case 'ś', 'š', 'ş', 'ŝ', 'ș':
		return 's'
	case 'ţ', 'ť', 'ŧ', 'ț':
		return 't'
	case 'ù', 'ú', 'û', 'ü', 'ũ', 'ū', 'ŭ', 'ů', 'ű', 'ų':
		return 'u'
	case 'ŵ':
		return 'w'
	case 'ý', 'ÿ', 'ŷ':
		return 'y'
	case 'ź', 'ž', 'ż':
		return 'z'
	case 'æ':
		return 'a'
	case 'œ':
		return 'o'
	case 'ß':
		return 's'
	}
	return r
}
