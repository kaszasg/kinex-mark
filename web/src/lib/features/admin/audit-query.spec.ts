// URL state and loading of the audit log (plan 8.7.9, 5.12).
import { readFileSync } from 'node:fs';
import { describe, expect, test, vi } from 'vitest';
import type { AuditEntry, AuditQuery } from '$lib/api/types.js';
import { hasMessage } from '$lib/i18n/index.js';
import {
	AUDIT_EVENTS,
	AUDIT_GROUPS,
	auditBaseQuery,
	auditFilterSearch,
	dayEnd,
	dayStart,
	DEFAULT_AUDIT_FILTER,
	eventPatterns,
	hasAuditFilters,
	isDay,
	loadAuditPage,
	parseAuditFilter
} from './audit-query.js';
import { makeAuditEntry } from './test-helpers.js';

const params = (search: string) => new URLSearchParams(search);

describe('audit events', () => {
	test('match the contract golden and are all translated', () => {
		const contract = JSON.parse(readFileSync('../internal/api/testdata/contract.json', 'utf8')) as {
			auditEvents: string[];
		};
		expect([...AUDIT_EVENTS]).toEqual(contract.auditEvents);
		for (const event of AUDIT_EVENTS) expect(hasMessage(`audit.events.${event}`)).toBe(true);
		for (const group of AUDIT_GROUPS)
			expect(hasMessage(`audit.filters.groups.${group.group}`)).toBe(true);
	});

	test('the privacy group spans two prefixes, the others one', () => {
		expect(eventPatterns('')).toEqual([]);
		expect(eventPatterns('auth.*')).toEqual(['auth.*']);
		expect(eventPatterns('privacy.*')).toEqual(['privacy.*', 'gdpr.*']);
		expect(eventPatterns('user.created')).toEqual(['user.created']);
	});
});

describe('audit filter', () => {
	test('defaults when the URL has nothing or rubbish', () => {
		expect(parseAuditFilter(params(''))).toEqual(DEFAULT_AUDIT_FILTER);
		expect(
			parseAuditFilter(
				params('event=evil.*&success=maybe&actorId=0&from=2026-02-30&to=yesterday&page=-1')
			)
		).toEqual(DEFAULT_AUDIT_FILTER);
	});

	test('round-trips every parameter', () => {
		const filter = {
			event: 'user.*',
			result: 'failure',
			actorId: 7,
			from: '2026-01-01',
			to: '2026-01-31',
			page: 2
		} as const;
		const search = auditFilterSearch(filter);
		expect(search).toBe(
			'?event=user.*&success=false&actorId=7&from=2026-01-01&to=2026-01-31&page=2'
		);
		expect(parseAuditFilter(params(search.slice(1)))).toEqual(filter);
		expect(parseAuditFilter(params('success=true&event=link.shared'))).toMatchObject({
			result: 'success',
			event: 'link.shared'
		});
		expect(auditFilterSearch(DEFAULT_AUDIT_FILTER)).toBe('');
		expect(auditFilterSearch({ ...DEFAULT_AUDIT_FILTER, result: 'success' })).toBe('?success=true');
	});

	test('knows when filters are set (the page does not count)', () => {
		expect(hasAuditFilters({ ...DEFAULT_AUDIT_FILTER, page: 2 })).toBe(false);
		for (const patch of [
			{ event: 'auth.*' },
			{ result: 'success' as const },
			{ actorId: 1 },
			{ from: '2026-01-01' },
			{ to: '2026-01-01' }
		])
			expect(hasAuditFilters({ ...DEFAULT_AUDIT_FILTER, ...patch })).toBe(true);
	});

	test('calendar days are validated', () => {
		expect(isDay('2026-02-28')).toBe(true);
		expect(isDay('2024-02-29')).toBe(true);
		expect(isDay('2026-02-29')).toBe(false);
		expect(isDay('2026-13-01')).toBe(false);
		expect(isDay('26-01-01')).toBe(false);
	});

	test('a day runs from local midnight to the last millisecond, in UTC', () => {
		const start = new Date(dayStart('2026-03-10'));
		const end = new Date(dayEnd('2026-03-10'));
		expect([start.getFullYear(), start.getMonth(), start.getDate(), start.getHours()]).toEqual([
			2026, 2, 10, 0
		]);
		expect([end.getDate(), end.getHours(), end.getMinutes(), end.getMilliseconds()]).toEqual([
			10, 23, 59, 999
		]);
		expect(dayStart('2026-03-10')).toMatch(/Z$/);
	});

	test('the base query carries actor, result and period', () => {
		expect(auditBaseQuery(DEFAULT_AUDIT_FILTER)).toEqual({
			actorId: undefined,
			success: undefined,
			from: undefined,
			to: undefined
		});
		expect(
			auditBaseQuery({
				...DEFAULT_AUDIT_FILTER,
				actorId: 3,
				result: 'success',
				from: '2026-01-01',
				to: '2026-01-02'
			})
		).toEqual({
			actorId: 3,
			success: true,
			from: dayStart('2026-01-01'),
			to: dayEnd('2026-01-02')
		});
		expect(auditBaseQuery({ ...DEFAULT_AUDIT_FILTER, result: 'failure' }).success).toBe(false);
	});
});

describe('loadAuditPage', () => {
	const at = (minute: number) => new Date(Date.UTC(2026, 0, 5, 10, minute)).toISOString();

	test('a single pattern is one request with the page offset', async () => {
		const items = [makeAuditEntry()];
		const list = vi.fn(async () => ({ items, total: 120, limit: 50, offset: 50 }));
		const result = await loadAuditPage(list, { ...DEFAULT_AUDIT_FILTER, event: 'auth.*', page: 2 });
		expect(result).toEqual({ items, total: 120 });
		expect(list).toHaveBeenCalledWith({
			actorId: undefined,
			success: undefined,
			from: undefined,
			to: undefined,
			event: 'auth.*',
			limit: 50,
			offset: 50
		});
	});

	test('no event filter sends no event', async () => {
		const list = vi.fn(async (query: AuditQuery) => ({
			items: [] as AuditEntry[],
			total: 0,
			limit: query.limit ?? 50,
			offset: 0
		}));
		await loadAuditPage(list, DEFAULT_AUDIT_FILTER);
		expect(list).toHaveBeenCalledTimes(1);
		expect(list.mock.calls[0]?.[0].event).toBeUndefined();
	});

	test('the privacy group merges two prefixes newest first and pages the merged list', async () => {
		const privacy = [5, 3, 1].map((m) =>
			makeAuditEntry({ id: 100 + m, at: at(m), event: 'privacy.consent_accepted' })
		);
		const gdpr = [4, 2].map((m) =>
			makeAuditEntry({ id: 200 + m, at: at(m), event: 'gdpr.export' })
		);
		const sources: Record<string, AuditEntry[]> = { 'privacy.*': privacy, 'gdpr.*': gdpr };
		const list = vi.fn(async (query: AuditQuery) => {
			const all = sources[query.event ?? ''];
			const offset = query.offset ?? 0;
			return {
				items: all.slice(offset, offset + (query.limit ?? 50)),
				total: all.length,
				limit: query.limit ?? 50,
				offset
			};
		});
		const first = await loadAuditPage(list, { ...DEFAULT_AUDIT_FILTER, event: 'privacy.*' }, 2);
		expect(first.total).toBe(5);
		expect(first.items.map((e) => e.id)).toEqual([105, 204]);
		const second = await loadAuditPage(
			list,
			{ ...DEFAULT_AUDIT_FILTER, event: 'privacy.*', page: 2 },
			2
		);
		expect(second.items.map((e) => e.id)).toEqual([103, 202]);
		const third = await loadAuditPage(
			list,
			{ ...DEFAULT_AUDIT_FILTER, event: 'privacy.*', page: 3 },
			2
		);
		expect(third.items.map((e) => e.id)).toEqual([101]);
	});

	test('equal times are ordered by id and deep pages fetch in chunks of 200', async () => {
		const same = at(1);
		const big = Array.from({ length: 450 }, (_, i) =>
			makeAuditEntry({ id: 1000 - i, at: same, event: 'privacy.notice_bumped' })
		);
		const list = vi.fn(async (query: AuditQuery) => {
			const all = query.event === 'privacy.*' ? big : [];
			const offset = query.offset ?? 0;
			return {
				items: all.slice(offset, offset + (query.limit ?? 50)),
				total: all.length,
				limit: query.limit ?? 50,
				offset
			};
		});
		const result = await loadAuditPage(list, {
			...DEFAULT_AUDIT_FILTER,
			event: 'privacy.*',
			page: 6
		});
		expect(result.total).toBe(450);
		expect(result.items.map((e) => e.id)).toEqual(big.slice(250, 300).map((e) => e.id));
		const privacyCalls = list.mock.calls.filter(([q]) => q.event === 'privacy.*');
		expect(privacyCalls.map(([q]) => [q.offset, q.limit])).toEqual([
			[0, 200],
			[200, 100]
		]);
		const gdprCalls = list.mock.calls.filter(([q]) => q.event === 'gdpr.*');
		expect(gdprCalls).toHaveLength(1);
	});
});
