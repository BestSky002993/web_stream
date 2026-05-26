// LocalStorage-backed JWT helpers.
//
// localStorage is acceptable for Phase 11 per spec. A future phase can
// swap this for an HttpOnly cookie (the call sites stay the same as
// long as we keep the shape `getToken / setToken / clearToken`).

const KEY = 'vsa.token';

export function getToken(): string | null {
	if (typeof window === 'undefined') return null;
	try {
		return window.localStorage.getItem(KEY);
	} catch {
		return null;
	}
}

export function setToken(token: string): void {
	if (typeof window === 'undefined') return;
	try {
		window.localStorage.setItem(KEY, token);
	} catch {
		// quota / private mode — swallow
	}
}

export function clearToken(): void {
	if (typeof window === 'undefined') return;
	try {
		window.localStorage.removeItem(KEY);
	} catch {
		// noop
	}
}

/** Decode the JWT payload (no signature check — server validates). */
export function decodeJWT(token: string): { exp?: number; sub?: string; email?: string; role?: string } | null {
	const parts = token.split('.');
	if (parts.length !== 3) return null;
	try {
		// base64url → base64
		const b64 = parts[1].replace(/-/g, '+').replace(/_/g, '/');
		const json = typeof atob === 'function' ? atob(b64) : Buffer.from(b64, 'base64').toString('utf-8');
		return JSON.parse(json);
	} catch {
		return null;
	}
}

/** True if the token has an exp claim in the past (best-effort, server is source of truth). */
export function isExpired(token: string): boolean {
	const payload = decodeJWT(token);
	if (!payload || typeof payload.exp !== 'number') return false;
	return Date.now() / 1000 > payload.exp;
}
