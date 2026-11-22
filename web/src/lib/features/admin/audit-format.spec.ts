// Translation of audit events and details (plan 8.5.8).
import { describe, expect, test } from 'vitest';
import { i18n, t } from '$lib/i18n/index.js';
import { detailRows, eventLabel, targetLabel } from './audit-format.js';

const rows = (event: string, details: Record<string, unknown>) =>
	Object.fromEntries(detailRows({ event, details }).map((row) => [row.key, row]));

describe('audit labels', () => {
	test('events and targets are translated, unknown codes stay raw', () => {
		expect(eventLabel('auth.login.failure')).toEqual({
			text: 'Sikertelen bejelentkezés',
			raw: false
		});
		expect(eventLabel('user.2fa_reset').text).toBe(t('audit.events.user.2fa_reset'));
		expect(eventLabel('robot.dance')).toEqual({ text: 'robot.dance', raw: true });
		expect(targetLabel('tag')).toEqual({ text: 'Címke', raw: false });
		expect(targetLabel('planet')).toEqual({ text: 'planet', raw: true });
	});

	test('labels follow the active language', () => {
		i18n.setLocale('de', { persist: false });
		expect(eventLabel('auth.logout').text).toBe('Abgemeldet');
		i18n.setLocale('en', { persist: false });
		expect(eventLabel('auth.logout').text).toBe('Signed out');
	});
});

describe('detail rows', () => {
	test('enumerated values are translated', () => {
		const r = rows('auth.login.failure', { reason: 'unknown_user' });
		expect(r.reason.label).toEqual({ text: 'Ok', raw: false });
		expect(r.reason.value).toEqual({ text: 'Ismeretlen felhasználónév', raw: false });
		const s = rows('auth.login.success', { method: 'recovery_code' });
		expect(s.method.value.text).toBe('Helyreállító kód');
		expect(rows('auth.login.locked', { scope: 'ip' }).scope.value.text).toBe('IP-cím');
		expect(rows('bootstrap.admin_created', { via: 'cli' }).via.value.text).toBe('Parancssor');
		expect(rows('link.shared', { targetType: 'all' }).targetType.value.text).toBe('Mindenki');
	});

	test('unknown keys and values stay raw', () => {
		const r = rows('auth.login.failure', { reason: 'cosmic_ray', mood: 'calm', blob: { a: 1 } });
		expect(r.reason.value).toEqual({ text: 'cosmic_ray', raw: true });
		expect(r.mood.label).toEqual({ text: 'mood', raw: true });
		expect(r.mood.value).toEqual({ text: 'calm', raw: true });
		expect(r.blob.value).toEqual({ text: '{"a":1}', raw: true });
		expect(rows('x', { reason: 3 }).reason.value).toEqual({ text: '3', raw: true });
	});

	test('role and active changes show both sides', () => {
		const r = rows('user.updated', {
			changed: ['role', 'active', 'displayName'],
			role: { from: 'developer', to: 'admin' },
			active: { from: true, to: false }
		});
		expect(r.changed.value.text).toBe('Szerepkör, Aktív, Megjelenített név');
		expect(r.role.value.text).toBe('Fejlesztő → Adminisztrátor');
		expect(r.active.value.text).toBe('Igen → Nem');
		expect(rows('user.updated', { role: { from: 'x', to: 'y' } }).role.value.raw).toBe(true);
	});

	test('single values: role, booleans, numbers, nulls', () => {
		const r = rows('user.created', {
			role: 'customer',
			generatedPassword: true,
			targetRole: null,
			remaining: 12480,
			sessionId: 12480,
			count: 'many',
			active: 'yes'
		});
		expect(r.role.value.text).toBe('Ügyfél');
		expect(r.generatedPassword.value.text).toBe('Igen');
		expect(r.targetRole.value).toEqual({ text: '–', raw: false });
		expect(r.remaining.value.text).toBe('12 480');
		expect(r.sessionId.value.text).toBe('12480');
		expect(r.count.value).toEqual({ text: 'many', raw: true });
		expect(r.active.value.raw).toBe(true);
		expect(rows('x', { role: 5 }).role.value.raw).toBe(true);
		expect(rows('x', { targetRole: 'developer' }).targetRole.value.text).toBe('Fejlesztő');
		expect(rows('x', { other: null }).other.value).toEqual({ text: '–', raw: false });
	});

	test('settings changes use the settings field labels', () => {
		const r = rows('settings.updated', { changed: ['defaultArchiveDays', 'mystery'] });
		expect(r.changed.value).toEqual({ text: 'Archiválási időszak, mystery', raw: false });
		expect(rows('settings.updated', { changed: ['nope'] }).changed.value.raw).toBe(true);
		expect(rows('settings.updated', { changed: 'role' }).changed.value.raw).toBe(true);
		expect(rows('settings.updated', { changed: [] }).changed.value).toEqual({
			text: '',
			raw: false
		});
	});

	test('no details give no rows', () => {
		expect(detailRows({ event: 'auth.logout', details: {} })).toEqual([]);
	});
});
