/**
 * Shared handling for form submission failures.
 *
 * The server reports `invalid_argument` violations per field; anything else is
 * shown as one form-level message. Screens assign both pieces back to their
 * local `$state` after a rejected request.
 */
import { fieldErrorsOf } from '#lib/api/errors';

export interface FormErrorOutcome {
	/** Field name → description for `invalid_argument` violations. */
	fieldErrors: Record<string, string>;
	/** Fallback form-level message, empty when field violations exist. */
	message: string;
}

/** Splits a rejected form request into field errors or one fallback message. */
export function formErrorOf(error: unknown, fallbackMessage: string): FormErrorOutcome {
	const fieldErrors = fieldErrorsOf(error);
	if (Object.keys(fieldErrors).length > 0) {
		return { fieldErrors, message: '' };
	}
	return { fieldErrors: {}, message: fallbackMessage };
}
