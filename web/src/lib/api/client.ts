// Typed fetch wrapper for the same-origin JSON API (plan 8.3).
import {
	ApiError,
	codeFromStatus,
	type AuthFailureCode,
	type ErrorCode,
	type FieldCode
} from './errors.js';

export type Query = Record<string, string | number | boolean | undefined | null>;
export type RequestOptions = {
	query?: Query;
	body?: unknown; // JSON-encoded
	raw?: BodyInit; // import: raw file body, with contentType
	contentType?: string;
	signal?: AbortSignal;
	keepalive?: boolean; // visit tracking
	skipAuthRedirect?: boolean; // login, me probe
};
export type Method = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';

type AuthFailureHandler = (code: AuthFailureCode) => void;

const AUTH_FAILURE_CODES: readonly ErrorCode[] = [
	'unauthenticated',
	'password_change_required',
	'consent_required'
];

let authFailureHandler: AuthFailureHandler | null = null;

/** Registers the single handler that reacts to session loss and open gates (the session store). */
export function onAuthFailure(handler: AuthFailureHandler | null): void {
	authFailureHandler = handler;
}

/** Builds the request URL: '/api' + path (or a root path such as /healthz) + the non-empty query values. */
export function buildUrl(path: string, query?: Query): string {
	const base = path.startsWith('/healthz') ? path : '/api' + path;
	if (!query) return base;
	const params = new URLSearchParams();
	for (const [key, value] of Object.entries(query)) {
		if (value === undefined || value === null || value === '') continue;
		params.append(key, String(value));
	}
	const search = params.toString();
	return search ? `${base}?${search}` : base;
}

function headersFor(opts: RequestOptions): Record<string, string> {
	const headers: Record<string, string> = {
		'X-Kinex-Request': '1',
		Accept: 'application/json'
	};
	if (opts.raw !== undefined) {
		headers['Content-Type'] = opts.contentType ?? 'application/octet-stream';
	} else if (opts.body !== undefined) {
		headers['Content-Type'] = opts.contentType ?? 'application/json';
	}
	return headers;
}

function isAbortError(err: unknown): boolean {
	return (
		typeof err === 'object' &&
		err !== null &&
		'name' in err &&
		(err as { name: unknown }).name === 'AbortError'
	);
}

async function send(method: Method, path: string, opts: RequestOptions): Promise<Response> {
	const init: RequestInit = {
		method,
		headers: headersFor(opts),
		credentials: 'same-origin',
		signal: opts.signal,
		keepalive: opts.keepalive
	};
	if (opts.raw !== undefined) init.body = opts.raw;
	else if (opts.body !== undefined) init.body = JSON.stringify(opts.body);

	let res: Response;
	try {
		res = await fetch(buildUrl(path, opts.query), init);
	} catch (err) {
		if (isAbortError(err)) throw err;
		throw new ApiError(0, 'network_error');
	}
	if (!res.ok) throw await toApiError(res, opts);
	return res;
}

type ErrorBody = {
	code?: unknown;
	fields?: unknown;
	details?: unknown;
};

function asRecord(value: unknown): Record<string, unknown> {
	return typeof value === 'object' && value !== null && !Array.isArray(value)
		? { ...(value as Record<string, unknown>) }
		: {};
}

async function toApiError(res: Response, opts: RequestOptions): Promise<ApiError> {
	const body = (await res.json().catch(() => null)) as ErrorBody | null;
	const code =
		body && typeof body.code === 'string' && body.code !== ''
			? (body.code as ErrorCode)
			: codeFromStatus(res.status);
	const fields = asRecord(body?.fields) as Record<string, FieldCode>;
	const details = asRecord(body?.details);
	if (res.status === 429 && details.retryAfterSeconds === undefined) {
		const retryAfter = Number.parseInt(res.headers.get('Retry-After') ?? '', 10);
		if (Number.isFinite(retryAfter)) details.retryAfterSeconds = retryAfter;
	}
	const err = new ApiError(res.status, code, fields, details);
	if (!opts.skipAuthRedirect && AUTH_FAILURE_CODES.includes(code) && authFailureHandler) {
		err.authHandled = true;
		authFailureHandler(code as AuthFailureCode);
	}
	return err;
}

/** Sends one API request and decodes the JSON answer; 204 and empty bodies give undefined. */
export async function request<T>(
	method: Method,
	path: string,
	opts: RequestOptions = {}
): Promise<T> {
	const res = await send(method, path, opts);
	if (res.status === 204) return undefined as T;
	const text = await res.text();
	if (text.trim() === '') return undefined as T;
	return JSON.parse(text) as T;
}

/** Reads the file name of a Content-Disposition header (filename*= first, then filename=). */
export function filenameFromDisposition(header: string | null, fallback = 'download'): string {
	if (!header) return fallback;
	const extended = /filename\*\s*=\s*([^']*)'[^']*'([^;]+)/i.exec(header);
	if (extended) {
		try {
			return decodeURIComponent(extended[2].trim());
		} catch {
			// Fall through to the plain parameter.
		}
	}
	const plain = /filename\s*=\s*("([^"]*)"|[^;]+)/i.exec(header);
	if (plain) return (plain[2] ?? plain[1]).trim();
	return fallback;
}

/** Downloads a file answer: the body as a Blob and the server-provided file name. */
export async function download(
	path: string,
	query?: Query
): Promise<{ blob: Blob; filename: string }> {
	const res = await send('GET', path, { query });
	const blob = await res.blob();
	return { blob, filename: filenameFromDisposition(res.headers.get('Content-Disposition')) };
}
