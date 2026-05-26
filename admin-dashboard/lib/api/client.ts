import { clearToken, getToken } from '@/lib/auth/storage';

export class APIError extends Error {
	status: number;
	body?: unknown;
	constructor(status: number, message: string, body?: unknown) {
		super(message);
		this.status = status;
		this.body = body;
	}
}

// Empty/omitted means same-origin. In production the dashboard is served
// behind nginx at the configured admin base path, and browser API calls
// must go back through nginx instead of trying to reach the internal API
// container port directly.
const BASE = process.env.NEXT_PUBLIC_API_BASE_URL ?? '';
const APP_BASE_PATH = process.env.NEXT_PUBLIC_BASE_PATH || '';

export function apiBase(): string {
	return BASE.replace(/\/$/, '');
}

type RequestOptions = Omit<RequestInit, 'body' | 'headers'> & {
	json?: unknown;
	body?: BodyInit;
	headers?: Record<string, string>;
	auth?: boolean; // default true
};

/**
 * Authenticated fetch wrapper.
 *
 * - Adds Authorization: Bearer <token> when `auth !== false`.
 * - Encodes JSON bodies when `json` is provided.
 * - Throws APIError for non-2xx responses (with the body parsed if JSON).
 * - On 401 (and `auth !== false`), clears the token and redirects to /login.
 */
export async function api<T = unknown>(path: string, opts: RequestOptions = {}): Promise<T> {
	const { json, body, headers = {}, auth = true, ...rest } = opts;
	const finalHeaders: Record<string, string> = { Accept: 'application/json', ...headers };

	let finalBody: BodyInit | undefined = body;
	if (json !== undefined) {
		finalHeaders['Content-Type'] = 'application/json';
		finalBody = JSON.stringify(json);
	}

	if (auth) {
		const token = getToken();
		if (token) finalHeaders['Authorization'] = `Bearer ${token}`;
	}

	const res = await fetch(apiBase() + path, {
		...rest,
		headers: finalHeaders,
		body: finalBody,
	});

	const text = await res.text();
	const parsed = text ? safeJSON(text) : undefined;

	if (!res.ok) {
		const msg = (parsed && typeof parsed === 'object' && 'error' in parsed) ? String((parsed as any).error) : `HTTP ${res.status}`;

		if (res.status === 401 && auth) {
			clearToken();
			if (typeof window !== 'undefined' && !window.location.pathname.startsWith(appPath('/login'))) {
				const next = encodeURIComponent(window.location.pathname + window.location.search);
				window.location.href = appPath(`/login?next=${next}`);
			}
		}
		throw new APIError(res.status, msg, parsed);
	}

	// 204 No Content
	if (res.status === 204) return undefined as T;
	return (parsed ?? undefined) as T;
}

function safeJSON(s: string): unknown {
	try {
		return JSON.parse(s);
	} catch {
		return undefined;
	}
}

/** SWR fetcher — first arg is the API path. */
export function swrFetcher<T = unknown>(path: string): Promise<T> {
	return api<T>(path, { auth: true });
}

function appPath(path: string): string {
	if (!APP_BASE_PATH) return path;
	return `${APP_BASE_PATH}${path}`;
}
