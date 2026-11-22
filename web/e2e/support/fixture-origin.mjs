// Local metadata fixture origin for the E2E suite (plan 9.4.1, step 3): a dependency-free
// node:http server with the pages and icons of e2e/fixtures/site/. The server under test
// fetches titles and icons from here (APP_ALLOW_LOOPBACK_FETCH=true).
//
// Standalone use, for debugging: node e2e/support/fixture-origin.mjs
import { createServer } from 'node:http';
import { readFile } from 'node:fs/promises';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { FIXTURE_HOST, FIXTURE_PORT } from './constants.mjs';

export const SITE_DIR = fileURLToPath(new URL('../fixtures/site/', import.meta.url));

/** How long /slow keeps the connection open before it answers (longer than APP_FETCH_TIMEOUT). */
export const SLOW_DELAY_MS = 10_000;

const UTF8_HTML = 'text/html; charset=utf-8';

/** Fixed routes: path -> file in the site directory and its Content-Type. */
const STATIC_ROUTES = new Map([
	['/alpha', { file: 'alpha.html', type: UTF8_HTML }],
	['/svg', { file: 'svg.html', type: UTF8_HTML }],
	['/plain', { file: 'plain.txt', type: 'text/plain; charset=utf-8' }],
	['/favicon.ico', { file: 'favicon.ico', type: 'image/x-icon' }],
	['/alpha.png', { file: 'alpha.png', type: 'image/png' }],
	['/icon.svg', { file: 'icon.svg', type: 'image/svg+xml' }]
]);

/** Unique pages: /p/<id> and their icon /p/<id>/icon.png. */
const UNIQUE_PAGE = /^\/p\/([A-Za-z0-9._-]{1,128})$/;
const UNIQUE_ICON = /^\/p\/([A-Za-z0-9._-]{1,128})\/icon\.png$/;

/**
 * Encodes a string as windows-1250. Node has no legacy encoder, so the table is the inverse
 * of the WHATWG decoder (needs the full ICU data that official Node builds ship).
 * @param {string} text
 * @returns {Buffer}
 */
export function encodeWindows1250(text) {
	const decoder = new TextDecoder('windows-1250');
	/** @type {Map<string, number>} */
	const table = new Map();
	for (let byte = 0x80; byte <= 0xff; byte++) {
		const char = decoder.decode(Uint8Array.of(byte));
		if (char !== '\ufffd' && !table.has(char)) table.set(char, byte);
	}
	const out = [];
	for (const char of text) {
		const code = char.codePointAt(0) ?? 0;
		if (code < 0x80) {
			out.push(code);
			continue;
		}
		const byte = table.get(char);
		if (byte === undefined) {
			throw new Error(`character U+${code.toString(16).toUpperCase()} has no windows-1250 byte`);
		}
		out.push(byte);
	}
	return Buffer.from(out);
}

/**
 * @param {import('node:http').ServerResponse} res
 * @param {number} status
 * @param {string} type
 * @param {Buffer | string} body
 * @param {boolean} head
 */
function send(res, status, type, body, head) {
	const buffer = typeof body === 'string' ? Buffer.from(body, 'utf8') : body;
	res.writeHead(status, {
		'Content-Type': type,
		'Content-Length': String(buffer.length),
		'Cache-Control': 'no-store'
	});
	res.end(head ? undefined : buffer);
}

/**
 * Starts the fixture origin.
 * @param {{ host?: string, port?: number, siteDir?: string, log?: (message: string) => void }} [options]
 * @returns {Promise<{ origin: string, close: () => Promise<void> }>}
 */
export async function startFixtureOrigin(options = {}) {
	const host = options.host ?? FIXTURE_HOST;
	const port = options.port ?? FIXTURE_PORT;
	const siteDir = options.siteDir ?? SITE_DIR;
	const log = options.log ?? (() => {});
	/** @type {Set<NodeJS.Timeout>} */
	const timers = new Set();

	/** @param {string} name */
	const readSite = (name) => readFile(new URL(name, pathToFileURL(siteDir)));

	const server = createServer(async (req, res) => {
		const head = req.method === 'HEAD';
		if (req.method !== 'GET' && !head) {
			send(res, 405, 'text/plain; charset=utf-8', 'method not allowed\n', false);
			return;
		}
		const path = new URL(req.url ?? '/', 'http://fixture.invalid').pathname;
		try {
			const route = STATIC_ROUTES.get(path);
			if (route) {
				send(res, 200, route.type, await readSite(route.file), head);
				return;
			}
			if (path === '/beta') {
				// Stored as UTF-8 for the editor, served as windows-1250 like a legacy site.
				const html = (await readSite('beta.html')).toString('utf8');
				send(res, 200, 'text/html; charset=windows-1250', encodeWindows1250(html), head);
				return;
			}
			if (path === '/slow') {
				const timer = setTimeout(() => {
					timers.delete(timer);
					send(res, 200, UTF8_HTML, '<!doctype html><title>Slow page</title>\n', head);
				}, SLOW_DELAY_MS);
				timers.add(timer);
				res.on('close', () => {
					clearTimeout(timer);
					timers.delete(timer);
				});
				return;
			}
			const page = UNIQUE_PAGE.exec(path);
			if (page) {
				const template = (await readSite('page.html')).toString('utf8');
				send(res, 200, UTF8_HTML, template.replaceAll('{{id}}', page[1]), head);
				return;
			}
			if (UNIQUE_ICON.test(path)) {
				send(res, 200, 'image/png', await readSite('page-icon.png'), head);
				return;
			}
			send(res, 404, 'text/plain; charset=utf-8', 'not found\n', head);
		} catch (err) {
			log(`fixture origin: ${path} failed: ${err instanceof Error ? err.message : String(err)}`);
			if (!res.headersSent) send(res, 500, 'text/plain; charset=utf-8', 'internal error\n', head);
		}
	});

	await new Promise((resolve, reject) => {
		server.once('error', reject);
		server.listen(port, host, () => {
			server.off('error', reject);
			resolve(undefined);
		});
	});

	return {
		origin: `http://${host}:${port}`,
		close: () =>
			new Promise((resolve) => {
				for (const timer of timers) clearTimeout(timer);
				timers.clear();
				server.close(() => resolve());
				server.closeAllConnections();
			})
	};
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
	const fixture = await startFixtureOrigin({ log: console.error });
	console.log(`fixture origin listening on ${fixture.origin}`);
	const stop = () => {
		fixture.close().then(() => process.exit(0));
	};
	process.once('SIGINT', stop);
	process.once('SIGTERM', stop);
}
