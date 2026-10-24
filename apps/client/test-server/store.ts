/**
 * In-memory fake store for the frontend-only Connect boundary.
 *
 * This is a real stateful implementation, not a mock: handlers read and write
 * it exactly as the Go server reads and writes SQLite, and tests reset it
 * between cases via `POST /__test/reset` (see `main.ts`).
 *
 * Password hashing is deliberately not bcrypt: bcrypt is a Go-side
 * implementation detail and the fake only needs credential equivalence. Node's
 * built-in scrypt keeps the double dependency-free.
 */
import { createHash, randomBytes, randomUUID, scryptSync, timingSafeEqual } from 'node:crypto';

/** Session cookie name, matching `server/internal/auth`. */
export const SESSION_COOKIE = 'mbc_session';

/** Absolute session lifetime (30 days), matching the Go contract. */
export const SESSION_LIFETIME_MS = 30 * 24 * 60 * 60 * 1000;

export interface FakeUser {
	id: string;
	email: string;
	name: string;
}

interface StoredUser extends FakeUser {
	passwordHash: string;
}

interface StoredSession {
	userId: string;
	expiresAt: number;
}

function hashToken(raw: string): string {
	return createHash('sha256').update(raw).digest('hex');
}

function hashPassword(password: string): string {
	const salt = randomBytes(16).toString('hex');
	const derived = scryptSync(password, salt, 32).toString('hex');
	return `${salt}:${derived}`;
}

function passwordMatches(password: string, stored: string): boolean {
	const [salt, expected] = stored.split(':');
	if (salt === undefined || expected === undefined) {
		return false;
	}
	const derived = scryptSync(password, salt, 32);
	const expectedBytes = Buffer.from(expected, 'hex');
	return derived.length === expectedBytes.length && timingSafeEqual(derived, expectedBytes);
}

/** A stateful in-memory stand-in for the server's SQLite-backed auth store. */
export class FakeStore {
	private readonly usersByEmail = new Map<string, StoredUser>();
	private readonly usersById = new Map<string, StoredUser>();
	private readonly sessions = new Map<string, StoredSession>();

	/** Clears every user and session; called between tests. */
	reset(): void {
		this.usersByEmail.clear();
		this.usersById.clear();
		this.sessions.clear();
	}

	findUserByEmail(email: string): FakeUser | undefined {
		return this.usersByEmail.get(email);
	}

	createUser(email: string, name: string, password: string): FakeUser {
		const user: StoredUser = {
			id: randomUUID(),
			email,
			name,
			passwordHash: hashPassword(password)
		};
		this.usersByEmail.set(email, user);
		this.usersById.set(user.id, user);
		return user;
	}

	/** Returns the user for a valid credential pair, or undefined. */
	authenticate(email: string, password: string): FakeUser | undefined {
		const user = this.usersByEmail.get(email);
		if (user === undefined || !passwordMatches(password, user.passwordHash)) {
			return undefined;
		}
		return user;
	}

	/** Mints an opaque session token for a user. Only its hash is retained. */
	createSession(userId: string, now: number = Date.now()): { token: string; expiresAt: number } {
		const token = randomBytes(32).toString('base64url');
		const expiresAt = now + SESSION_LIFETIME_MS;
		this.sessions.set(hashToken(token), { userId, expiresAt });
		return { token, expiresAt };
	}

	/** Resolves a raw cookie token to its user, deleting it when expired. */
	findSessionUser(token: string, now: number = Date.now()): FakeUser | undefined {
		const hash = hashToken(token);
		const session = this.sessions.get(hash);
		if (session === undefined) {
			return undefined;
		}
		if (session.expiresAt <= now) {
			this.sessions.delete(hash);
			return undefined;
		}
		return this.usersById.get(session.userId);
	}

	deleteSession(token: string): void {
		this.sessions.delete(hashToken(token));
	}
}
