/**
 * Fake AuthService for the frontend-only Connect seam.
 *
 * The behavior mirrors `server/internal/auth`: the `mbc_session` cookie is a
 * 30-day absolute opaque session (HttpOnly, SameSite=Lax, Path=/), duplicate
 * registration is `already_exists`/`email_taken`, a bad email or password is
 * the same `unauthenticated`/`invalid_credentials`, and `Me` without a valid
 * session is `unauthenticated`/`session_expired`. Validation failures are
 * `invalid_argument` with field paths.
 *
 * The CSRF posture (`X-Requested-With` + same-origin Origin) lives in the HTTP
 * wrapper in `main.ts`, exactly where the Go middleware sits.
 */
import { create } from '@bufbuild/protobuf';
import { Code, ConnectError, type HandlerContext, type ServiceImpl } from '@connectrpc/connect';

import { SESSION_COOKIE, SESSION_LIFETIME_MS } from './store.ts';
import type { FakeStore, FakeUser } from './store.ts';
import { AuthService } from '../../../packages/shared/src/gen/mybadcoach/auth/v1/auth_pb.ts';
import { ErrorDetailSchema } from '../../../packages/shared/src/gen/mybadcoach/errors/v1/errors_pb.ts';

/** The `AuthService` methods implemented by the fake. */
export type AuthImplementation = Partial<ServiceImpl<typeof AuthService>>;

interface FieldViolationInput {
	field: string;
	description: string;
}

const SESSION_MAX_AGE_SECONDS = String(SESSION_LIFETIME_MS / 1000);

function connectError(
	code: Code,
	machineCode: string,
	message: string,
	violations: FieldViolationInput[] = []
): ConnectError {
	return new ConnectError(message, code, undefined, [
		{
			desc: ErrorDetailSchema,
			value: create(ErrorDetailSchema, { code: machineCode, fieldViolations: violations })
		}
	]);
}

function normalizeEmail(email: string): string {
	return email.trim().toLowerCase();
}

function validEmail(email: string): boolean {
	return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}

function validateRegistration(
	email: string,
	name: string,
	password: string
): FieldViolationInput[] {
	const violations: FieldViolationInput[] = [];
	if (!validEmail(email)) {
		violations.push({ field: 'email', description: 'must be a valid email address' });
	}
	if (name === '') {
		violations.push({ field: 'name', description: 'must not be empty' });
	}
	if (Array.from(password).length < 8) {
		violations.push({ field: 'password', description: 'must be at least 8 characters' });
	} else if (Buffer.byteLength(password, 'utf8') > 72) {
		violations.push({ field: 'password', description: 'must be at most 72 bytes' });
	}
	return violations;
}

/** Reads the raw session token from a Connect request's `Cookie` header. */
function sessionToken(header: Headers): string | undefined {
	const cookieHeader = header.get('cookie');
	if (cookieHeader === null) {
		return undefined;
	}
	for (const part of cookieHeader.split(';')) {
		const separator = part.indexOf('=');
		if (separator === -1) {
			continue;
		}
		if (part.slice(0, separator).trim() === SESSION_COOKIE) {
			return part.slice(separator + 1);
		}
	}
	return undefined;
}

function sessionCookie(token: string, expiresAt: number): string {
	const attributes = [
		`${SESSION_COOKIE}=${token}`,
		'Path=/',
		'HttpOnly',
		'SameSite=Lax',
		`Max-Age=${SESSION_MAX_AGE_SECONDS}`,
		`Expires=${new Date(expiresAt).toUTCString()}`
	];
	return attributes.join('; ');
}

function clearedSessionCookie(): string {
	return [
		`${SESSION_COOKIE}=`,
		'Path=/',
		'HttpOnly',
		'SameSite=Lax',
		'Max-Age=0',
		'Expires=Thu, 01 Jan 1970 00:00:00 GMT'
	].join('; ');
}

function issueSession(store: FakeStore, context: HandlerContext, userId: string): void {
	const { token, expiresAt } = store.createSession(userId);
	context.responseHeader.append('Set-Cookie', sessionCookie(token, expiresAt));
}

function invalidCredentials(): ConnectError {
	return connectError(
		Code.Unauthenticated,
		'invalid_credentials',
		'invalid email or password'
	);
}

function sessionExpired(): ConnectError {
	return connectError(Code.Unauthenticated, 'session_expired', 'session expired');
}

function userMessage(user: FakeUser): { id: string; email: string; name: string } {
	return { id: user.id, email: user.email, name: user.name };
}

/** Builds the fake AuthService implementation over a store. */
export function createAuthImplementation(store: FakeStore): AuthImplementation {
	return {
		register(request, context) {
			const email = normalizeEmail(request.email);
			const name = request.name.trim();
			const violations = validateRegistration(email, name, request.password);
			if (violations.length > 0) {
				throw connectError(
					Code.InvalidArgument,
					'invalid_argument',
					'invalid registration',
					violations
				);
			}
			if (store.findUserByEmail(email) !== undefined) {
				throw connectError(Code.AlreadyExists, 'email_taken', 'email already registered');
			}

			const user = store.createUser(email, name, request.password);
			issueSession(store, context, user.id);
			return { user: userMessage(user) };
		},

		login(request, context) {
			const user = store.authenticate(normalizeEmail(request.email), request.password);
			if (user === undefined) {
				throw invalidCredentials();
			}
			issueSession(store, context, user.id);
			return { user: userMessage(user) };
		},

		logout(_request, context) {
			const token = sessionToken(context.requestHeader);
			if (token !== undefined) {
				store.deleteSession(token);
			}
			context.responseHeader.append('Set-Cookie', clearedSessionCookie());
			return {};
		},

		me(_request, context) {
			const token = sessionToken(context.requestHeader);
			const user = token === undefined ? undefined : store.findSessionUser(token);
			if (user === undefined) {
				throw sessionExpired();
			}
			return { user: userMessage(user) };
		}
	};
}
