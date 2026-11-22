import { render, screen, within } from '@testing-library/svelte';
import { modeStorageKey, themeStorageKey } from 'mode-watcher';
import { afterEach, describe, expect, test } from 'vitest';
import { LOCALE_STORAGE_KEY } from '$lib/i18n/detect.js';
import { fmt, i18n, LOCALES, t } from '$lib/i18n/index.js';
import { makePublicConfig } from '$lib/test/fixtures.js';
import PrivacyNotice from './PrivacyNotice.svelte';

const SECTIONS = [
	'controller',
	'data',
	'purposes',
	'legalBasis',
	'recipients',
	'transfers',
	'automatedDecisions',
	'retention',
	'storage',
	'fetching',
	'adminAccess',
	'rights'
] as const;

const config = (partial: Partial<ReturnType<typeof makePublicConfig>['privacy']> = {}) => ({
	...makePublicConfig().privacy,
	...partial
});

const section = (name: string): HTMLElement =>
	document.querySelector<HTMLElement>(`[data-section="${name}"]`) as HTMLElement;

describe('PrivacyNotice', () => {
	test('renders the twelve sections of the notice in order with h2 headings', () => {
		render(PrivacyNotice, { config: config() });
		const headings = screen.getAllByRole('heading', { level: 2 });
		expect(headings.map((h) => h.textContent?.trim())).toEqual(
			SECTIONS.map((s) => t(`privacy.sections.${s}.title`))
		);
		for (const s of SECTIONS.filter((s) => s !== 'retention')) {
			expect(section(s)).toHaveTextContent(t(`privacy.sections.${s}.body`));
		}
	});

	test('lists the controller data and marks empty values', () => {
		render(PrivacyNotice, {
			config: config({
				controllerName: 'Kinexium Kft.',
				controllerAddress: '',
				controllerEmail: 'privacy@kinexium.example',
				dpoContact: '  '
			})
		});
		const controller = within(section('controller'));
		expect(controller.getByText('Kinexium Kft.')).toBeInTheDocument();
		expect(controller.getByRole('link', { name: 'privacy@kinexium.example' })).toHaveAttribute(
			'href',
			'mailto:privacy@kinexium.example'
		);
		expect(controller.getAllByText(fmt.missing)).toHaveLength(2);
		for (const label of ['name', 'address', 'email', 'dpo'] as const) {
			expect(controller.getByText(t(`privacy.controller.${label}`))).toBeInTheDocument();
		}
	});

	test('an empty e-mail is shown as missing, not as a link', () => {
		render(PrivacyNotice, { config: config({ controllerEmail: '' }) });
		expect(within(section('controller')).queryByRole('link')).toBeNull();
	});

	test('the retention periods come from the configuration', () => {
		render(PrivacyNotice, {
			config: config({ sessionAbsoluteHours: 24, sessionIdleMinutes: 30, auditRetentionDays: 365 })
		});
		const expected = t('privacy.sections.retention.body', {
			hours: t('common.units.hours', { count: 24 }),
			minutes: t('common.units.minutes', { count: 30 }),
			days: t('common.units.days', { count: 365 })
		});
		expect(section('retention').querySelector('p')?.textContent?.trim()).toBe(expected);
	});

	test('lists the seven rights and names the generic authority when none is set', () => {
		render(PrivacyNotice, { config: config({ supervisoryAuthority: '' }) });
		const items = within(section('rights')).getAllByRole('listitem');
		expect(items).toHaveLength(7);
		for (const right of [
			'access',
			'portability',
			'rectification',
			'erasure',
			'restriction',
			'objection'
		] as const) {
			expect(section('rights')).toHaveTextContent(t(`privacy.rights.${right}`));
		}
		expect(items[6]).toHaveTextContent(
			t('privacy.rights.complaint', { authority: t('privacy.rights.defaultAuthority') })
		);
	});

	test('names the configured supervisory authority', () => {
		render(PrivacyNotice, {
			config: config({ supervisoryAuthority: 'Nemzeti Adatvédelmi Hatóság, naih.hu' })
		});
		expect(section('rights')).toHaveTextContent('Nemzeti Adatvédelmi Hatóság, naih.hu');
	});

	test('the compact variant uses h3 headings and keeps its markers', () => {
		const { container } = render(PrivacyNotice, {
			config: config({ noticeVersion: 4 }),
			compact: true
		});
		expect(screen.getAllByRole('heading', { level: 3 })).toHaveLength(SECTIONS.length);
		expect(screen.queryAllByRole('heading', { level: 2 })).toHaveLength(0);
		const root = container.querySelector('[data-compact="true"]');
		expect(root).toHaveAttribute('data-version', '4');
	});

	test('follows the language switch', async () => {
		render(PrivacyNotice, { config: config() });
		i18n.setLocale('de', { persist: false });
		expect(await screen.findByRole('heading', { name: 'Deine Rechte' })).toBeInTheDocument();
	});

	describe('accuracy of the notice (GDPR Art. 13, plan 6.14, 8.7.10)', () => {
		afterEach(() => i18n.setLocale('hu', { persist: false }));

		test.each(LOCALES)('%s: the storage section names every key the browser keeps', (locale) => {
			i18n.setLocale(locale, { persist: false });
			const body = t('privacy.sections.storage.body');
			for (const key of [modeStorageKey.current, themeStorageKey.current, LOCALE_STORAGE_KEY]) {
				expect(body).toContain(key);
			}
			expect(body).toContain('kxm_session');
		});

		test.each([
			['hu', ['felhasználónevedet', 'minden aktív felhasználót']],
			['en', ['your username', 'every active user']],
			['de', ['deinen Benutzernamen', 'jeden aktiven Benutzer']]
		] as const)(
			'%s: the recipients section names the username and the user directory',
			(locale, phrases) => {
				i18n.setLocale(locale, { persist: false });
				const body = t('privacy.sections.recipients.body');
				for (const phrase of phrases) expect(body).toContain(phrase);
			}
		);
	});
});
