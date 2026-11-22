// URL state and loading of the audit log (plan 8.7.9, 5.12).
import type { AuditEntry, AuditQuery, Page } from '$lib/api/types.js';
import { ADMIN_PAGE_SIZE, pageOffset, parsePage } from './paging.js';

/** Every audit event code, in the order of plan 6.10 (the contract golden checks it). */
export const AUDIT_EVENTS = [
	'bootstrap.admin_created',
	'auth.login.success',
	'auth.login.failure',
	'auth.login.locked',
	'auth.logout',
	'auth.recovery_code.used',
	'auth.password.changed',
	'auth.2fa.enabled',
	'auth.2fa.disabled',
	'auth.2fa.recovery_regenerated',
	'session.revoked',
	'session.revoked_others',
	'user.created',
	'user.updated',
	'user.deactivated',
	'user.activated',
	'user.deleted',
	'user.password_reset',
	'user.2fa_reset',
	'user.unlocked',
	'user.sessions_revoked',
	'user.self_deleted',
	'settings.updated',
	'privacy.notice_bumped',
	'privacy.consent_accepted',
	'gdpr.export',
	'link.shared',
	'link.unshared'
] as const;

export type AuditGroup = 'auth' | 'session' | 'user' | 'settings' | 'privacy' | 'link';

/**
 * Event groups of the filter. The value is what the URL carries; `patterns` are the server
 * filters. The privacy group spans two prefixes, so it is merged from two requests.
 */
export const AUDIT_GROUPS: readonly { group: AuditGroup; value: string; patterns: string[] }[] = [
	{ group: 'auth', value: 'auth.*', patterns: ['auth.*'] },
	{ group: 'session', value: 'session.*', patterns: ['session.*'] },
	{ group: 'user', value: 'user.*', patterns: ['user.*'] },
	{ group: 'settings', value: 'settings.*', patterns: ['settings.*'] },
	{ group: 'privacy', value: 'privacy.*', patterns: ['privacy.*', 'gdpr.*'] },
	{ group: 'link', value: 'link.*', patterns: ['link.*'] }
];

export type AuditResultFilter = '' | 'success' | 'failure';

export type AuditFilter = {
	/** '' (every event), a group value ('auth.*') or one event code. */
	event: string;
	result: AuditResultFilter;
	actorId: number | null;
	/** Local calendar days, YYYY-MM-DD, or ''. */
	from: string;
	to: string;
	page: number;
};

export const DEFAULT_AUDIT_FILTER: AuditFilter = {
	event: '',
	result: '',
	actorId: null,
	from: '',
	to: '',
	page: 1
};

/** The server returns at most this many rows per request (plan 5.4). */
export const AUDIT_MAX_LIMIT = 200;

const knownEvent = (value: string): boolean =>
	AUDIT_GROUPS.some((g) => g.value === value) ||
	(AUDIT_EVENTS as readonly string[]).includes(value);

/** True for a real calendar day in YYYY-MM-DD form. */
export function isDay(value: string): boolean {
	const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
	if (!match) return false;
	const [year, month, day] = [Number(match[1]), Number(match[2]), Number(match[3])];
	const date = new Date(year, month - 1, day);
	return date.getFullYear() === year && date.getMonth() === month - 1 && date.getDate() === day;
}

/** Reads the filter from the query string; unknown or malformed values are dropped. */
export function parseAuditFilter(params: Pick<URLSearchParams, 'get'>): AuditFilter {
	const event = params.get('event') ?? '';
	const result = params.get('success');
	const actor = params.get('actorId') ?? '';
	const from = params.get('from') ?? '';
	const to = params.get('to') ?? '';
	return {
		event: knownEvent(event) ? event : '',
		result: result === 'true' ? 'success' : result === 'false' ? 'failure' : '',
		actorId: /^[1-9]\d{0,15}$/.test(actor) ? Number(actor) : null,
		from: isDay(from) ? from : '',
		to: isDay(to) ? to : '',
		page: parsePage(params.get('page'))
	};
}

/** The query string of a filter ('' for the default view). */
export function auditFilterSearch(filter: AuditFilter): string {
	const params = new URLSearchParams();
	if (filter.event) params.set('event', filter.event);
	if (filter.result) params.set('success', filter.result === 'success' ? 'true' : 'false');
	if (filter.actorId !== null) params.set('actorId', String(filter.actorId));
	if (filter.from) params.set('from', filter.from);
	if (filter.to) params.set('to', filter.to);
	if (filter.page > 1) params.set('page', String(filter.page));
	const search = params.toString();
	return search ? `?${search}` : '';
}

/** True when any filter differs from the default view (the page number does not count). */
export function hasAuditFilters(filter: AuditFilter): boolean {
	return (
		filter.event !== '' ||
		filter.result !== '' ||
		filter.actorId !== null ||
		filter.from !== '' ||
		filter.to !== ''
	);
}

function localDay(day: string): Date {
	const [year, month, date] = day.split('-').map(Number);
	return new Date(year, month - 1, date);
}

/** The start of a local calendar day as an RFC 3339 UTC instant. */
export function dayStart(day: string): string {
	return localDay(day).toISOString();
}

/** The last millisecond of a local calendar day as an RFC 3339 UTC instant. */
export function dayEnd(day: string): string {
	const next = localDay(day);
	next.setDate(next.getDate() + 1);
	return new Date(next.getTime() - 1).toISOString();
}

/** The server event filters of a filter value: [] means no event filter. */
export function eventPatterns(event: string): string[] {
	if (!event) return [];
	return AUDIT_GROUPS.find((g) => g.value === event)?.patterns ?? [event];
}

/** The query parameters shared by every request of a filter (without event and paging). */
export function auditBaseQuery(filter: AuditFilter): AuditQuery {
	return {
		actorId: filter.actorId ?? undefined,
		success: filter.result === '' ? undefined : filter.result === 'success',
		from: filter.from ? dayStart(filter.from) : undefined,
		to: filter.to ? dayEnd(filter.to) : undefined
	};
}

type ListAudit = (query: AuditQuery) => Promise<Page<AuditEntry>>;

/** Newest first; the id breaks ties, as on the server. */
function newestFirst(a: AuditEntry, b: AuditEntry): number {
	const diff = Date.parse(b.at) - Date.parse(a.at);
	return diff !== 0 ? diff : b.id - a.id;
}

/** The first `count` entries of one event pattern, in requests of at most AUDIT_MAX_LIMIT rows. */
async function head(
	list: ListAudit,
	query: AuditQuery,
	count: number
): Promise<{ items: AuditEntry[]; total: number }> {
	const items: AuditEntry[] = [];
	let total = 0;
	do {
		const limit = Math.min(AUDIT_MAX_LIMIT, count - items.length);
		const page = await list({ ...query, limit, offset: items.length });
		total = page.total;
		items.push(...page.items);
		if (page.items.length < limit) break;
	} while (items.length < count && items.length < total);
	return { items, total };
}

/**
 * Loads one page of the audit log. A single event pattern is one request; several patterns
 * (the privacy group) are merged newest first: each pattern contributes its first
 * offset + size rows, and the merged list is cut to the page.
 */
export async function loadAuditPage(
	list: ListAudit,
	filter: AuditFilter,
	size = ADMIN_PAGE_SIZE
): Promise<{ items: AuditEntry[]; total: number }> {
	const base = auditBaseQuery(filter);
	const offset = pageOffset(filter.page, size);
	const patterns = eventPatterns(filter.event);
	if (patterns.length <= 1) {
		const page = await list({ ...base, event: patterns[0], limit: size, offset });
		return { items: page.items, total: page.total };
	}
	const parts = await Promise.all(
		patterns.map((event) => head(list, { ...base, event }, offset + size))
	);
	const merged = parts.flatMap((part) => part.items).sort(newestFirst);
	return {
		items: merged.slice(offset, offset + size),
		total: parts.reduce((sum, part) => sum + part.total, 0)
	};
}
