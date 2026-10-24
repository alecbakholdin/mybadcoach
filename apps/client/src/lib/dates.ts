/**
 * Display helper for a date range. Dates are ISO-8601 calendar dates
 * (`YYYY-MM-DD`); an empty string means "unset".
 */

/** Joins non-empty start/end dates with an en dash, or returns ''. */
export function dateRange(start: string, end: string): string {
	if (start !== '' && end !== '') {
		return `${start} – ${end}`;
	}
	return start !== '' ? start : end;
}
