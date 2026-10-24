/**
 * Typed access to the machine-readable details the server attaches to Connect
 * errors (`mybadcoach.errors.v1.ErrorDetail`, see `.scratch/backend-api/issues/04`).
 *
 * Screens use these helpers to tell an expired session (`session_expired`) from
 * a bad credential pair (`invalid_credentials`) and to map `invalid_argument`
 * field violations onto form fields without depending on error message text.
 */
import { Code, ConnectError } from '@connectrpc/connect';

import { ErrorDetailSchema, type ErrorDetail } from '$shared/gen/mybadcoach/errors/v1/errors_pb';

/** Returns the decoded `ErrorDetail`, when the error carries one. */
export function errorDetailOf(error: unknown): ErrorDetail | undefined {
	if (!(error instanceof ConnectError)) {
		return undefined;
	}
	return error.findDetails(ErrorDetailSchema)[0];
}

/** Returns the stable machine code (`email_taken`, `invalid_credentials`, ...). */
export function machineCodeOf(error: unknown): string | undefined {
	return errorDetailOf(error)?.code;
}

/** Maps `invalid_argument` field violations onto field name → description. */
export function fieldErrorsOf(error: unknown): Record<string, string> {
	const fieldErrors: Record<string, string> = {};
	for (const violation of errorDetailOf(error)?.fieldViolations ?? []) {
		fieldErrors[violation.field] = violation.description;
	}
	return fieldErrors;
}

/** True for a `not_found` Connect error (missing entity or no membership). */
export function isNotFound(error: unknown): boolean {
	return error instanceof ConnectError && error.code === Code.NotFound;
}

/**
 * Awaits a request that may legitimately target a missing entity, mapping
 * `not_found` to `undefined` so route loads can render their not-found state.
 * Every other error is rethrown.
 */
export async function undefinedIfNotFound<T>(promise: Promise<T>): Promise<T | undefined> {
	try {
		return await promise;
	} catch (error) {
		if (isNotFound(error)) {
			return undefined;
		}
		throw error;
	}
}

/**
 * True for the shared bad-email-or-password error. The server returns an
 * `unauthenticated` Connect code for both `invalid_credentials` and
 * `session_expired`; only the former belongs on the login form.
 */
export function isInvalidCredentials(error: unknown): boolean {
	return (
		error instanceof ConnectError &&
		error.code === Code.Unauthenticated &&
		machineCodeOf(error) === 'invalid_credentials'
	);
}
