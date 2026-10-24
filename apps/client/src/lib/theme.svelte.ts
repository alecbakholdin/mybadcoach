/**
 * Theme state for the Night-game token system.
 *
 * A rune module (not a store) so components read `theme.current` reactively.
 * `app.html` applies a persisted theme before first paint; this module adopts
 * that value and keeps the DOM attribute and localStorage in sync on toggle.
 */

export type Theme = 'dark' | 'light';

/** Must match the inline boot script in `src/app.html`. */
export const THEME_STORAGE_KEY = 'mbc-theme';

function readInitial(): Theme {
	if (typeof document !== 'undefined' && document.documentElement.getAttribute('data-theme') === 'light') {
		return 'light';
	}
	return 'dark';
}

let current = $state<Theme>(readInitial());

function apply(next: Theme): void {
	current = next;
	document.documentElement.setAttribute('data-theme', next);
	try {
		localStorage.setItem(THEME_STORAGE_KEY, next);
	} catch {
		// Storage can be unavailable (private mode); the theme still applies.
	}
}

export const theme = {
	get current(): Theme {
		return current;
	},
	toggle(): void {
		apply(current === 'dark' ? 'light' : 'dark');
	}
};
