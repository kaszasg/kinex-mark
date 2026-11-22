// The typed fetch wrapper (plan 8.3, 9.3).
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { apiError, json, mockApi, noContent } from '$lib/test/mock-api.js';
import { buildUrl, download, filenameFromDisposition, onAuthFailure, request } from './client.js';
import { ApiError, codeFromStatus, isApiError } from './errors.js';

const handler = vi.fn();

beforeEach(() => {
	onAuthFailure(handler);
});

afterEach(() => {
	onAuthFailure(null);
	handler.mockReset();
	vi.unstubAllGlobals();
});

describe('buildUrl', () => {
	test('prefixes /api and drops empty query values', () => {
		expect(buildUrl('/links')).toBe('/api/links');
		expect(
			buildUrl('/links', { q: 'a b', tags: '', limit: 50, x: undefined, y: null, untagged: true })
		).toBe('/api/links?q=a+b&limit=50&untagged=true');
		expect(buildUrl('/links', { q: '' })).toBe('/api/links');
	});

	test('keeps root paths such as /healthz', () => {
		expect(buildUrl('/healthz')).toBe('/healthz');
	});
});

describe('request', () => {
	test('sends the CSRF and accept headers with same-origin credentials', async () => {
		const api = mockApi({ 'GET /api/tags': json({ items: [] }) });
		await expect(request('GET', '/tags')).resolves.toEqual({ items: [] });
		const req = api.requests[0];
		expect(req.headers.get('X-Kinex-Request')).toBe('1');
		expect(req.headers.get('Accept')).toBe('application/json');
		expect(req.headers.get('Content-Type')).toBeNull();
		expect(req.init.credentials).toBe('same-origin');
	});

	test('JSON-encodes a body', async () => {
		const api = mockApi({ 'POST /api/tags': json({ id: 1 }, 201) });
		await request('POST', '/tags', { body: { name: 'infra' } });
		expect(api.requests[0].headers.get('Content-Type')).toBe('application/json');
		expect(api.requests[0].body).toEqual({ name: 'infra' });
	});

	test('sends a raw body with its content type', async () => {
		const api = mockApi({ 'POST /api/links/import': json({ created: 1 }) });
		await request('POST', '/links/import', { raw: '<html></html>', contentType: 'text/html' });
		expect(api.requests[0].headers.get('Content-Type')).toBe('text/html');
		expect(api.requests[0].rawBody).toBe('<html></html>');
		await request('POST', '/links/import', { raw: 'x' });
		expect(api.requests[1].headers.get('Content-Type')).toBe('application/octet-stream');
	});

	test('204 and empty bodies decode to undefined', async () => {
		mockApi({ 'DELETE /api/tags/1': noContent(), 'POST /api/x': { status: 200, body: '' } });
		await expect(request('DELETE', '/tags/1')).resolves.toBeUndefined();
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => new Response('  ', { status: 200 }))
		);
		await expect(request('POST', '/x')).resolves.toBeUndefined();
	});

	test('maps an error body to ApiError with fields and details', async () => {
		mockApi({
			'POST /api/links': apiError(400, 'validation_failed', { url: 'invalid_format' }, { a: 1 })
		});
		const err = await request('POST', '/links', { body: {} }).catch((e: unknown) => e);
		expect(isApiError(err)).toBe(true);
		const apiErr = err as ApiError;
		expect(apiErr.status).toBe(400);
		expect(apiErr.code).toBe('validation_failed');
		expect(apiErr.field('url')).toBe('invalid_format');
		expect(apiErr.field('title')).toBeUndefined();
		expect(apiErr.details).toEqual({ a: 1 });
		expect(apiErr.is('validation_failed', 'forbidden')).toBe(true);
		expect(apiErr.is('forbidden')).toBe(false);
		expect(apiErr.name).toBe('ApiError');
		expect(handler).not.toHaveBeenCalled();
	});

	test('falls back to the status code when the body has no code', async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => new Response('<html>bad gateway</html>', { status: 502 }))
		);
		await expect(request('GET', '/links')).rejects.toMatchObject({
			status: 502,
			code: 'internal_error'
		});
		vi.stubGlobal(
			'fetch',
			vi.fn(
				async () => new Response(JSON.stringify({ fields: [1], details: 'x' }), { status: 404 })
			)
		);
		await expect(request('GET', '/links/9')).rejects.toMatchObject({
			code: 'not_found',
			fields: {},
			details: {}
		});
	});

	test('reads Retry-After on 429 when the body carries no wait', async () => {
		mockApi({
			'POST /api/auth/login': apiError(429, 'too_many_attempts', undefined, undefined, {
				'Retry-After': '90'
			})
		});
		await expect(request('POST', '/auth/login', { skipAuthRedirect: true })).rejects.toMatchObject({
			details: { retryAfterSeconds: 90 }
		});
		mockApi({
			'POST /api/auth/login': apiError(
				429,
				'too_many_attempts',
				undefined,
				{ retryAfterSeconds: 30 },
				{ 'Retry-After': '90' }
			)
		});
		await expect(request('POST', '/auth/login')).rejects.toMatchObject({
			details: { retryAfterSeconds: 30 }
		});
		mockApi({ 'POST /api/auth/login': apiError(429, 'too_many_attempts') });
		await expect(request('POST', '/auth/login')).rejects.toMatchObject({ details: {} });
	});

	test.each(['unauthenticated', 'password_change_required', 'consent_required'] as const)(
		'%s calls the auth failure handler',
		async (code) => {
			mockApi({ 'GET /api/links': apiError(code === 'unauthenticated' ? 401 : 403, code) });
			await expect(request('GET', '/links')).rejects.toMatchObject({ authHandled: true });
			expect(handler).toHaveBeenCalledWith(code);
		}
	);

	test('skipAuthRedirect suppresses the handler', async () => {
		mockApi({ 'GET /api/auth/me': apiError(401, 'unauthenticated') });
		await expect(request('GET', '/auth/me', { skipAuthRedirect: true })).rejects.toMatchObject({
			authHandled: false
		});
		expect(handler).not.toHaveBeenCalled();
	});

	test('without a handler nothing is called', async () => {
		onAuthFailure(null);
		mockApi({ 'GET /api/links': apiError(401, 'unauthenticated') });
		await expect(request('GET', '/links')).rejects.toBeInstanceOf(ApiError);
		expect(handler).not.toHaveBeenCalled();
	});

	test('a network failure becomes network_error', async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => {
				throw new TypeError('Failed to fetch');
			})
		);
		await expect(request('GET', '/links')).rejects.toMatchObject({
			status: 0,
			code: 'network_error'
		});
	});

	test('an abort is rethrown unchanged', async () => {
		const abort = new DOMException('aborted', 'AbortError');
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => {
				throw abort;
			})
		);
		await expect(request('GET', '/links')).rejects.toBe(abort);
	});

	test('passes signal and keepalive through', async () => {
		const api = mockApi({ 'POST /api/links/1/visit': json({ id: 1 }) });
		const controller = new AbortController();
		await request('POST', '/links/1/visit', { keepalive: true, signal: controller.signal });
		expect(api.requests[0].init.keepalive).toBe(true);
		expect(api.requests[0].init.signal).toBe(controller.signal);
	});
});

describe('download', () => {
	test('returns the blob and the server file name', async () => {
		mockApi({
			'GET /api/links/export': {
				status: 200,
				body: new Blob(['{}'], { type: 'application/json' }),
				headers: { 'Content-Disposition': 'attachment; filename="kinex-mark-links.json"' }
			}
		});
		const result = await download('/links/export', { format: 'json' });
		expect(result.filename).toBe('kinex-mark-links.json');
		expect(await result.blob.text()).toBe('{}');
	});

	test('falls back to a generic name', async () => {
		mockApi({ 'GET /api/me/export': { status: 200, body: new Blob(['x']) } });
		expect((await download('/me/export')).filename).toBe('download');
	});
});

describe('filenameFromDisposition', () => {
	test.each([
		[null, 'download'],
		['attachment', 'download'],
		['attachment; filename="a b.json"', 'a b.json'],
		['attachment; filename=plain.html', 'plain.html'],
		['attachment; filename*=UTF-8\'\'%C3%A9les.json; filename="x.json"', 'éles.json'],
		['attachment; filename*=UTF-8\'\'%E0%A4%A; filename="fallback.json"', 'fallback.json']
	])('%s -> %s', (header, expected) => {
		expect(filenameFromDisposition(header)).toBe(expected);
	});

	test('custom fallback', () => {
		expect(filenameFromDisposition(null, 'export.json')).toBe('export.json');
	});
});

describe('codeFromStatus', () => {
	test.each([
		[401, 'unauthenticated'],
		[403, 'forbidden'],
		[404, 'not_found'],
		[413, 'payload_too_large'],
		[429, 'too_many_attempts'],
		[500, 'internal_error'],
		[418, 'internal_error']
	] as const)('%d -> %s', (status, code) => {
		expect(codeFromStatus(status)).toBe(code);
	});

	test('isApiError rejects other values', () => {
		expect(isApiError(new Error('x'))).toBe(false);
		expect(isApiError(new ApiError(0, 'network_error'))).toBe(true);
	});
});
