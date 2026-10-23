package structure

import "fmt"

// maxSlugAttempts bounds collision resolution for a single create.
const maxSlugAttempts = 1000

// slugTaken reports whether candidate is already used in the relevant scope.
type slugTaken func(candidate string) (bool, error)

// uniqueSlug builds the scoped-unique slug for a new entity. It starts from
// the name-derived base slug (or fallback, typically the new id, when the name
// has no ASCII letters or digits) and appends -2, -3, ... until a free slug is
// found.
func uniqueSlug(base, fallback string, taken slugTaken) (string, error) {
	if base == "" {
		base = fallback
	}
	for suffix := 1; suffix <= maxSlugAttempts; suffix++ {
		candidate := base
		if suffix > 1 {
			candidate = fmt.Sprintf("%s-%d", base, suffix)
		}
		isTaken, err := taken(candidate)
		if err != nil {
			return "", fmt.Errorf("check slug %q: %w", candidate, err)
		}
		if !isTaken {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no available slug for %q", base)
}
