// Privacy notice card of Settings > Privacy (plan 8.7.6). The F1 notice is replaced by a double.
import { screen, within } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, test, vi } from 'vitest';
import type { PrivacyStatus } from '$lib/api/types.js';
import { fmt, t } from '$lib/i18n/index.js';
import { makeMe, makePublicConfig } from '$lib/test/fixtures.js';
import { apiError, json, mockApi } from '$lib/test/mock-api.js';
import { renderWithApp } from '$lib/test/render.js';
import ConsentHistory, { sortConsents } from './ConsentHistory.svelte';

vi.mock('$lib/features/privacy/PrivacyNotice.svelte', async () => ({
	default: (await import('./test-doubles/PrivacyNoticeDouble.svelte')).default
}));

const STATUS: PrivacyStatus = {
	currentVersion: 3,
	consentRequired: false,
	consents: [
		{ kind: 'privacy_notice', version: 1, acceptedAt: '2026-01-05T10:00:00.000Z' },
		{ kind: 'privacy_notice', version: 3, acceptedAt: '2026-11-02T08:30:00.000Z' }
	],
	ownSharedLinkCount: 0
};

const CONFIG = makePublicConfig({
	privacy: {
		...makePublicConfig().privacy,
		noticeVersion: 3,
		updatedAt: '2026-11-01T12:00:00.000Z'
	}
});

function spaced(text: string): string {
	return text.replace(/\s+/g, ' ');
}

afterEach(() => {
	vi.unstubAllGlobals();
});

describe('sortConsents', () => {
	test('newest version first, then the later acceptance', () => {
		const sorted = sortConsents([
			{ kind: 'privacy_notice', version: 2, acceptedAt: '2026-02-01T00:00:00Z' },
			{ kind: 'privacy_notice', version: 4, acceptedAt: '2026-03-01T00:00:00Z' },
			{ kind: 'privacy_notice', version: 2, acceptedAt: '2026-02-05T00:00:00Z' }
		]);
		expect(sorted.map((c) => `${c.version}@${c.acceptedAt.slice(5, 10)}`)).toEqual([
			'4@03-01',
			'2@02-05',
			'2@02-01'
		]);
	});
});

describe('ConsentHistory', () => {
	test('shows the current version, the acceptances and the notice text', async () => {
		mockApi({
			'GET /api/me/privacy': json(STATUS),
			'GET /api/public/config': json(CONFIG)
		});
		renderWithApp(ConsentHistory, {}, { me: makeMe() });
		const card = screen.getByRole('region', { name: t('security.privacy.notice.title') });
		expect(await within(card).findByTestId('notice-version')).toHaveTextContent('3');
		expect(card).toHaveTextContent(spaced(fmt.dateTime('2026-11-01T12:00:00.000Z')));

		const rows = within(card).getAllByRole('listitem');
		expect(rows).toHaveLength(2);
		expect(rows[0]).toHaveTextContent(t('security.privacy.notice.version', { version: 3 }));
		expect(rows[0]).toHaveTextContent(t('security.privacy.notice.currentBadge'));
		expect(rows[0]).toHaveTextContent(
			spaced(
				t('security.privacy.notice.acceptedAt', { time: fmt.dateTime('2026-11-02T08:30:00.000Z') })
			)
		);
		expect(rows[1]).toHaveTextContent(t('security.privacy.notice.version', { version: 1 }));
		expect(rows[1]).not.toHaveTextContent(t('security.privacy.notice.currentBadge'));

		const notice = within(card).getByTestId('privacy-notice');
		expect(notice).toHaveAttribute('data-compact', 'true');
		expect(notice).toHaveAttribute('data-version', '3');
		const region = within(card).getByRole('region', { name: t('security.privacy.notice.summary') });
		expect(region).toContainElement(notice);
		// The scrolling viewport itself is the region and takes the keyboard focus (axe
		// scrollable-region-focusable): the notice inside has nothing focusable.
		expect(region).toHaveAttribute('data-slot', 'scroll-area-viewport');
		expect(region).toHaveAttribute('tabindex', '0');
		expect(
			within(card).getByRole('link', { name: t('security.privacy.notice.open') })
		).toHaveAttribute('href', '/privacy');
	});

	test('no acceptance yet: empty state', async () => {
		mockApi({
			'GET /api/me/privacy': json({ ...STATUS, consents: [], consentRequired: true }),
			'GET /api/public/config': json(CONFIG)
		});
		renderWithApp(ConsentHistory, {}, { me: makeMe() });
		expect(await screen.findByText(t('security.privacy.notice.none.title'))).toBeInTheDocument();
		expect(screen.getByText(t('security.privacy.notice.none.text'))).toBeInTheDocument();
	});

	test('a failed load shows an alert and retries', async () => {
		const api = mockApi({
			'GET /api/me/privacy': apiError(500, 'internal_error'),
			'GET /api/public/config': json(CONFIG)
		});
		renderWithApp(ConsentHistory, {}, { me: makeMe() });
		const alert = await screen.findByRole('alert');
		expect(alert).toHaveTextContent(t('errors.internal_error.title'));
		api.route('GET /api/me/privacy', json(STATUS));
		await userEvent.click(within(alert).getByRole('button', { name: t('common.actions.retry') }));
		expect(await screen.findByTestId('notice-version')).toHaveTextContent('3');
		expect(api.calls('GET /api/me/privacy')).toHaveLength(2);
	});

	test('shows loading rows first', async () => {
		let release: () => void = () => undefined;
		mockApi({
			'GET /api/me/privacy': () =>
				new Promise((resolve) => {
					release = () => resolve(json(STATUS));
				}),
			'GET /api/public/config': json(CONFIG)
		});
		renderWithApp(ConsentHistory, {}, { me: makeMe() });
		expect(await screen.findAllByTestId('loading-row')).toHaveLength(3);
		release();
		expect(await screen.findByTestId('notice-version')).toBeInTheDocument();
	});
});
