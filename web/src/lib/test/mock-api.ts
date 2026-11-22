// fetch stub for API tests: "METHOD /path" -> response (plan 9.3.1).
import { vi } from 'vitest';
import type { ErrorCode, FieldCode } from '$lib/api/errors.js';
import { apiErrorBody } from './fixtures.js';

export type MockRequest = {
	method: string;
	path: string;
	url: URL;
	headers: Headers;
	body: unknown;
	rawBody: BodyInit | null | undefined;
	init: RequestInit;
};

export type MockResponse = {
	status?: number;
	body?: unknown;
	headers?: Record<string, string>;
};

export type MockRoute = MockResponse | ((req: MockRequest) => MockResponse | Promise<MockResponse>);

/** A JSON answer. */
export function json(
	body: unknown,
	status = 200,
	headers: Record<string, string> = {}
): MockResponse {
	return { status, body, headers };
}

/** An empty 204 answer. */
export function noContent(): MockResponse {
	return { status: 204 };
}

/** An error answer with the contract's body shape. */
export function apiError(
	status: number,
	code: ErrorCode,
	fields?: Record<string, FieldCode>,
	details?: Record<string, unknown>,
	headers: Record<string, string> = {}
): MockResponse {
	return { status, body: apiErrorBody(code, fields, details), headers };
}

async function toResponse(res: MockResponse): Promise<Response> {
	const status = res.status ?? 200;
	const headers = new Headers(res.headers);
	if (status === 204 || res.body === undefined) return new Response(null, { status, headers });
	if (res.body instanceof Blob) {
		// jsdom's Blob is not the Blob of Node's fetch: pass the bytes instead.
		if (res.body.type && !headers.has('Content-Type')) headers.set('Content-Type', res.body.type);
		return new Response(await res.body.text(), { status, headers });
	}
	if (!headers.has('Content-Type')) headers.set('Content-Type', 'application/json; charset=utf-8');
	return new Response(JSON.stringify(res.body), { status, headers });
}

function parseBody(body: BodyInit | null | undefined): unknown {
	if (typeof body !== 'string') return body ?? undefined;
	try {
		return JSON.parse(body);
	} catch {
		return body;
	}
}

/**
 * Installs a fetch stub. Keys are "METHOD /path" without the query string; an unmatched
 * request answers 404 endpoint_not_found. Returns the mock so tests can inspect the calls.
 */
export function mockApi(routes: Record<string, MockRoute> = {}) {
	const requests: MockRequest[] = [];
	const fetchMock = vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
		const url = new URL(String(input), 'http://localhost');
		const method = (init.method ?? 'GET').toUpperCase();
		const req: MockRequest = {
			method,
			path: url.pathname,
			url,
			headers: new Headers(init.headers),
			body: parseBody(init.body),
			rawBody: init.body,
			init
		};
		requests.push(req);
		const route = routes[`${method} ${url.pathname}`];
		if (!route) return toResponse(apiError(404, 'endpoint_not_found'));
		const res = typeof route === 'function' ? await route(req) : route;
		return toResponse(res);
	});
	vi.stubGlobal('fetch', fetchMock);
	return {
		fetch: fetchMock,
		requests,
		/** The requests that hit one "METHOD /path" key. */
		calls(key: string): MockRequest[] {
			return requests.filter((req) => `${req.method} ${req.path}` === key);
		},
		/** Adds or replaces a route after installation. */
		route(key: string, route: MockRoute) {
			routes[key] = route;
		}
	};
}
