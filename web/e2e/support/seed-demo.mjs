// Loads the demo data set (e2e/fixtures/demo-data.json) into a running server through the API
// (plan 9.7): users with fictional names, tags in every colour, links to public documentation
// sites (so they get icons), Hungarian descriptions, shares, pins, visits and archived links.
// Used for the exploratory GUI test and the manual screenshots. Dependency-free Node script.
//
//   npm run seed:demo -- --base http://127.0.0.1:8080 --password <admin password>
//
// Options: --base <url> (default http://127.0.0.1:8080), --password <admin password>
// (required), --admin-user <name> (default admin), --data <file> (default the bundled set).
// Re-running is safe: existing users, tags and links are reused and their state is left alone.
import { readFile } from 'node:fs/promises';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { parseArgs } from 'node:util';

const DEFAULT_DATA = fileURLToPath(new URL('../fixtures/demo-data.json', import.meta.url));

/** @param {string} message */
function log(message) {
	console.log(`[seed-demo] ${message}`);
}

/** A cookie-keeping JSON client for one signed-in user. */
export class ApiClient {
	/** @param {string} base */
	constructor(base) {
		this.base = new URL(base);
		this.cookies = new Map();
	}

	/**
	 * @param {string} method
	 * @param {string} path
	 * @param {unknown} [body]
	 * @returns {Promise<{ status: number, body: any }>}
	 */
	async call(method, path, body) {
		/** @type {Record<string, string>} */
		const headers = {
			'X-Kinex-Request': '1',
			Accept: 'application/json',
			Origin: this.base.origin
		};
		if (body !== undefined) headers['Content-Type'] = 'application/json';
		if (this.cookies.size > 0) {
			headers.Cookie = [...this.cookies].map(([name, value]) => `${name}=${value}`).join('; ');
		}
		const response = await fetch(new URL(path, this.base), {
			method,
			headers,
			body: body === undefined ? undefined : JSON.stringify(body),
			redirect: 'manual'
		});
		for (const header of response.headers.getSetCookie()) {
			const [pair, ...attributes] = header.split(';');
			const index = pair.indexOf('=');
			if (index < 1) continue;
			const name = pair.slice(0, index).trim();
			const value = pair.slice(index + 1).trim();
			const expired = attributes.some((attribute) =>
				/^\s*max-age\s*=\s*(-\d+|0)\s*$/i.test(attribute)
			);
			if (expired || value === '') this.cookies.delete(name);
			else this.cookies.set(name, value);
		}
		const text = await response.text();
		let parsed;
		try {
			parsed = text === '' ? undefined : JSON.parse(text);
		} catch {
			parsed = text;
		}
		return { status: response.status, body: parsed };
	}

	/**
	 * Like call(), but throws unless the status is one of the expected ones.
	 * @param {string} method
	 * @param {string} path
	 * @param {unknown} body
	 * @param {number[]} expected
	 */
	async expect(method, path, body, expected) {
		const result = await this.call(method, path, body);
		if (!expected.includes(result.status)) {
			const detail =
				typeof result.body === 'object' ? JSON.stringify(result.body) : String(result.body);
			throw new Error(`${method} ${path}: HTTP ${result.status} ${detail}`);
		}
		return result;
	}

	/**
	 * @param {string} username
	 * @param {string} password
	 */
	async signIn(username, password) {
		const result = await this.expect('POST', '/api/auth/login', { username, password }, [200]);
		if (result.body?.status !== 'ok') {
			throw new Error(
				`sign in as ${username}: status "${result.body?.status}" (2FA is not supported here)`
			);
		}
		return result.body.user;
	}

	async acceptPrivacyNotice() {
		const { body } = await this.expect('GET', '/api/me/privacy', undefined, [200]);
		if (body.consentRequired) {
			await this.expect('POST', '/api/me/privacy/consent', { version: body.currentVersion }, [204]);
		}
	}
}

/**
 * @param {ApiClient} admin
 * @param {{ username: string, displayName: string, role: string, password: string }} user
 * @param {string} base
 * @returns {Promise<{ id: number, client: ApiClient }>}
 */
async function ensureUser(admin, user, base) {
	const created = await admin.call('POST', '/api/admin/users', {
		username: user.username,
		displayName: user.displayName,
		role: user.role
	});
	const client = new ApiClient(base);
	if (created.status === 201) {
		await client.signIn(user.username, created.body.temporaryPassword);
		await client.expect(
			'POST',
			'/api/auth/password',
			{ currentPassword: created.body.temporaryPassword, newPassword: user.password },
			[204]
		);
		await client.acceptPrivacyNotice();
		log(`created user ${user.username} (${user.role})`);
		return { id: created.body.user.id, client };
	}
	if (created.status === 409 && created.body?.code === 'username_taken') {
		const me = await client.signIn(user.username, user.password);
		await client.acceptPrivacyNotice();
		log(`user ${user.username} already exists, reusing it`);
		return { id: me.id, client };
	}
	throw new Error(
		`create user ${user.username}: HTTP ${created.status} ${JSON.stringify(created.body)}`
	);
}

/**
 * @param {ApiClient} client
 * @param {{ name: string, color: string }[]} tags
 * @returns {Promise<Map<string, number>>}
 */
async function ensureTags(client, tags) {
	const ids = new Map();
	for (const tag of tags) {
		const result = await client.call('POST', '/api/tags', tag);
		if (result.status === 201) ids.set(tag.name, result.body.id);
		else if (result.status === 409 && result.body?.code === 'tag_duplicate') {
			ids.set(tag.name, result.body.details.tagId);
		} else {
			throw new Error(`create tag: HTTP ${result.status} ${JSON.stringify(result.body)}`);
		}
	}
	return ids;
}

/**
 * @param {Map<string, number>} tagIds
 * @param {string[] | undefined} names
 */
function resolveTags(tagIds, names = []) {
	return names.map((name) => {
		const id = tagIds.get(name);
		if (id === undefined) throw new Error(`unknown demo tag: ${name}`);
		return id;
	});
}

/**
 * @param {ApiClient} client
 * @param {number} linkId
 * @param {number} count
 */
async function visit(client, linkId, count) {
	for (let i = 0; i < count; i++) {
		await client.expect('POST', `/api/links/${linkId}/visit`, undefined, [200]);
	}
}

export async function main(argv = process.argv.slice(2)) {
	const { values } = parseArgs({
		args: argv,
		options: {
			base: { type: 'string', default: 'http://127.0.0.1:8080' },
			password: { type: 'string' },
			'admin-user': { type: 'string', default: 'admin' },
			data: { type: 'string', default: DEFAULT_DATA }
		}
	});
	if (!values.password) {
		throw new Error('missing --password <admin password>');
	}
	const base = values.base;
	const data = JSON.parse(await readFile(values.data, 'utf8'));
	if (data.format !== 'kinex-mark-demo/1')
		throw new Error(`unknown demo data format: ${data.format}`);

	const admin = new ApiClient(base);
	const adminMe = await admin.signIn(values['admin-user'], values.password);
	if (adminMe.mustChangePassword) {
		throw new Error('the admin must change its password first: sign in once in the browser');
	}
	await admin.acceptPrivacyNotice();

	/** @type {Map<string, { id: number, client: ApiClient }>} */
	const users = new Map();
	for (const user of data.users) {
		if (user.bootstrap) {
			await admin.expect('PATCH', '/api/me/profile', { displayName: user.displayName }, [200]);
			users.set(user.username, { id: adminMe.id, client: admin });
			continue;
		}
		users.set(user.username, await ensureUser(admin, user, base));
	}
	/** @param {string} name */
	const userOf = (name) => {
		const user = users.get(name);
		if (!user) throw new Error(`unknown demo user: ${name}`);
		return user;
	};

	/** @type {Map<string, Map<string, number>>} */
	const tagIds = new Map();
	for (const [owner, tags] of Object.entries(data.tags)) {
		tagIds.set(owner, await ensureTags(userOf(owner).client, tags));
	}

	/** @type {Map<string, number>} */
	const linkIds = new Map();
	/** Links created in this run; state is only applied to these, so a re-run adds nothing. */
	const fresh = new Set();
	let created = 0;
	for (const link of data.links) {
		const { client } = userOf(link.owner);
		const result = await client.call('POST', '/api/links', {
			url: link.url,
			description: link.description ?? '',
			tagIds: resolveTags(tagIds.get(link.owner) ?? new Map(), link.tags),
			pinned: link.pinned ?? false
		});
		if (result.status === 409 && result.body?.code === 'link_duplicate') {
			linkIds.set(link.url, result.body.details.linkId);
			continue;
		}
		if (result.status !== 201) {
			throw new Error(
				`create link ${link.url}: HTTP ${result.status} ${JSON.stringify(result.body)}`
			);
		}
		const id = result.body.id;
		linkIds.set(link.url, id);
		fresh.add(link.url);
		created += 1;
		await visit(client, id, link.visits ?? 0);
		for (const share of link.shares ?? []) {
			const target =
				share.targetType === 'user'
					? { targetType: 'user', targetUserId: userOf(share.user).id }
					: share.targetType === 'role'
						? { targetType: 'role', targetRole: share.role }
						: { targetType: 'all' };
			await client.expect('POST', `/api/links/${id}/shares`, target, [201, 409]);
		}
		if (link.archived) await client.expect('POST', `/api/links/${id}/archive`, undefined, [200]);
	}
	log(`created ${created} of ${data.links.length} links`);

	for (const state of data.recipientState ?? []) {
		const { client } = userOf(state.user);
		const id = linkIds.get(state.url);
		if (id === undefined) throw new Error(`recipient state for an unknown link: ${state.url}`);
		if (!fresh.has(state.url)) continue;
		if (state.tags?.length) {
			const tags = resolveTags(tagIds.get(state.user) ?? new Map(), state.tags);
			await client.expect('PUT', `/api/links/${id}/tags`, { tagIds: tags }, [200]);
		}
		if (state.pinned) await client.expect('POST', `/api/links/${id}/pin`, undefined, [200]);
		await visit(client, id, state.visits ?? 0);
	}

	for (const user of users.values()) {
		if (user.client !== admin) await user.client.call('POST', '/api/auth/logout');
	}
	await admin.call('POST', '/api/auth/logout');
	log('done; demo passwords are in e2e/fixtures/demo-data.json');
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
	try {
		await main();
	} catch (err) {
		console.error(`[seed-demo] ${err instanceof Error ? err.message : String(err)}`);
		process.exit(1);
	}
}
