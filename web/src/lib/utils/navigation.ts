// Redirect target validation (plan 8.7.0).

// U+0000-U+001F and U+007F.
const CONTROL = /[\u0000-\u001f\u007f]/;

/**
 * Accepts a `next` target only when it stays on the given origin. A backslash or a control
 * character rejects the value, because browsers normalise "/\evil.example" to "//evil.example".
 * Returns pathname + search + hash, otherwise "/".
 */
export function safeNext(next: string | null | undefined, origin: string): string {
	if (!next || CONTROL.test(next) || next.includes('\\')) return '/';
	let url: URL;
	try {
		url = new URL(next, origin);
	} catch {
		return '/';
	}
	if (url.origin !== origin) return '/';
	// A path that decodes to a second leading slash or a backslash would leave the origin later.
	let decoded = url.pathname;
	try {
		decoded = decodeURIComponent(url.pathname);
	} catch {
		return '/';
	}
	if (decoded.startsWith('//') || decoded.includes('\\')) return '/';
	return url.pathname + url.search + url.hash;
}

/** The login URL that brings the user back to the given path afterwards. */
export function loginUrl(path: string, reason?: 'expired' | 'deleted'): string {
	const params = new URLSearchParams();
	if (path && path !== '/') params.set('next', path);
	if (reason) params.set('reason', reason);
	const query = params.toString();
	return query ? `/login?${query}` : '/login';
}
