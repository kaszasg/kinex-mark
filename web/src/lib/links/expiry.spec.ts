import { describe, expect, it } from 'vitest';
import { makeLink } from '$lib/test/fixtures.js';
import { expiryInfo, showsExpiry } from './expiry.js';

const DAY = 86_400_000;
const start = new Date('2026-01-01T00:00:00Z');
const due = new Date(start.getTime() + 90 * DAY);

function at(daysAfterStart: number): Date {
	return new Date(start.getTime() + daysAfterStart * DAY);
}

function info(now: Date, dueAt: string | null = due.toISOString(), since = start.toISOString()) {
	return expiryInfo({ archiveDueAt: dueAt, inactiveSince: since, now });
}

describe('expiryInfo', () => {
	it('returns null without a due date or with an invalid one', () => {
		expect(info(at(0), null)).toBeNull();
		expect(info(at(0), 'nonsense')).toBeNull();
	});

	it('fills all five segments right after the last use', () => {
		expect(info(at(0))).toEqual({ segments: 5, daysLeft: 90, soon: false });
	});

	it('rounds the segments up', () => {
		expect(info(at(36))?.segments).toBe(3); // 54/90 * 5 = 3
		expect(info(at(37))?.segments).toBe(3); // 53/90 * 5 = 2.94
		expect(info(at(54))?.segments).toBe(2);
	});

	it('keeps at least one segment until the due date', () => {
		expect(info(at(89.9))).toMatchObject({ segments: 1, daysLeft: 1, soon: true });
	});

	it('turns to soon at seven days', () => {
		expect(info(at(83))).toMatchObject({ daysLeft: 7, soon: true });
		expect(info(at(82))).toMatchObject({ daysLeft: 8, soon: false });
	});

	it('reports zero when due or overdue', () => {
		expect(info(at(90))).toEqual({ segments: 0, daysLeft: 0, soon: true });
		expect(info(at(120))).toEqual({ segments: 0, daysLeft: 0, soon: true });
	});

	it('fills every segment when the start is unknown or after the due date', () => {
		expect(info(at(0), due.toISOString(), 'nonsense')?.segments).toBe(5);
		expect(info(at(0), due.toISOString(), at(100).toISOString())?.segments).toBe(5);
	});
});

describe('showsExpiry', () => {
	const dueAt = due.toISOString();

	it('shows on an active link with a due date', () => {
		expect(showsExpiry(makeLink({ archiveDueAt: dueAt }), 90)).toBe(true);
	});

	it('hides on pinned, archived, without a due date and when archiving is off', () => {
		expect(showsExpiry(makeLink({ archiveDueAt: dueAt, pinned: true }), 90)).toBe(false);
		expect(showsExpiry(makeLink({ archiveDueAt: dueAt, archivedAt: dueAt }), 90)).toBe(false);
		expect(showsExpiry(makeLink({ archiveDueAt: null }), 90)).toBe(false);
		expect(showsExpiry(makeLink({ archiveDueAt: dueAt }), 0)).toBe(false);
	});
});
