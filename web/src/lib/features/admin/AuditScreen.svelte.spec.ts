// The /admin/audit screen: states, retention subtitle, URL filters and paging (plan 8.7.9).
import { fireEvent, screen, within } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { goto } from '$app/navigation';
import { ApiError } from '$lib/api/errors.js';
import type { AuditEntry } from '$lib/api/types.js';
import { errorTitle, t } from '$lib/i18n/index.js';
import { publicConfig } from '$lib/state/public-config.svelte.js';
import { setTestPage } from '$lib/test/app-state.svelte.js';
import { makeMe, makePublicConfig } from '$lib/test/fixtures.js';
import { apiError, json, mockApi, type MockRequest } from '$lib/test/mock-api.js';
import { renderWithApp } from '$lib/test/render.js';
import AuditScreen from './AuditScreen.svelte';
import {
	findFloating,
	makeAdminUser,
	makeAuditEntry,
	makeSettingsResponse,
	page,
	user
} from './test-helpers.js';
import { normalizeSpace } from '$lib/test/text.js';

const me = makeMe('admin', { id: 1, username: 'kanna' });

function auditRoute(entries: AuditEntry[], total = entries.length) {
	// The privacy group asks twice (privacy.* and gdpr.*); only the first prefix has entries here.
	return (req: MockRequest) =>
		req.url.searchParams.get('event') === 'gdpr.*'
			? json(page([], 0))
			: json(page(entries, total, Number(req.url.searchParams.get('offset') ?? 0)));
}

const baseRoutes = {
	'GET /api/admin/settings': json(makeSettingsResponse({ auditRetentionDays: 180 })),
	'GET /api/admin/users': json(
		page([makeAdminUser({ id: 1, username: 'kanna' }), makeAdminUser({ id: 2, username: 'peter' })])
	)
};

function render(path = '/admin/audit') {
	setTestPage(path);
	return renderWithApp(AuditScreen, {}, { me });
}

beforeEach(() => {
	vi.mocked(goto).mockReset();
	vi.mocked(goto).mockImplementation(async (url) => setTestPage(String(url)));
});

afterEach(() => {
	publicConfig.config = null;
	vi.unstubAllGlobals();
});

describe('AuditScreen states', () => {
	test('loading, then the entries with the retention in the subtitle', async () => {
		let release: () => void = () => undefined;
		const gate = new Promise<void>((resolve) => (release = resolve));
		mockApi({
			...baseRoutes,
			'GET /api/admin/audit': async () => {
				await gate;
				return json(page([makeAuditEntry({ event: 'auth.logout' })]));
			}
		});
		render();
		expect(screen.getByRole('heading', { name: t('audit.pageTitle') })).toBeInTheDocument();
		expect(screen.getByRole('status')).toHaveTextContent(t('common.states.loading'));
		expect(
			await screen.findByText(normalizeSpace(t('audit.subtitle', { days: 180 })))
		).toBeInTheDocument();
		release();
		expect(await screen.findByText(t('audit.events.auth.logout'))).toBeInTheDocument();
		expect(screen.getByRole('search', { name: t('audit.filters.label') })).toBeInTheDocument();
	});

	test('without settings the retention comes from the public config', async () => {
		publicConfig.config = makePublicConfig();
		const days = publicConfig.config.privacy.auditRetentionDays;
		mockApi({
			'GET /api/admin/settings': apiError(500, 'internal_error'),
			'GET /api/admin/audit': auditRoute([])
		});
		render();
		expect(
			await screen.findByText(normalizeSpace(t('audit.subtitle', { days })))
		).toBeInTheDocument();
	});

	test('an error replaces the table and retry loads again', async () => {
		let fail = true;
		mockApi({
			...baseRoutes,
			'GET /api/admin/audit': () =>
				fail ? apiError(500, 'internal_error') : json(page([makeAuditEntry()]))
		});
		render();
		const alert = await screen.findByRole('alert');
		expect(alert).toHaveTextContent(errorTitle(new ApiError(500, 'internal_error')));
		fail = false;
		await user().click(within(alert).getByRole('button', { name: t('common.actions.retry') }));
		expect(await screen.findByText(t('audit.events.auth.login.success'))).toBeInTheDocument();
	});

	test('empty without filters: no clear button', async () => {
		mockApi({ ...baseRoutes, 'GET /api/admin/audit': auditRoute([]) });
		render();
		expect(await screen.findByText(t('audit.empty.title'))).toBeInTheDocument();
		expect(screen.queryByRole('button', { name: t('audit.filters.clear') })).toBeNull();
	});

	test('empty with filters: both clear buttons reset the URL', async () => {
		mockApi({ ...baseRoutes, 'GET /api/admin/audit': auditRoute([]) });
		render('/admin/audit?success=false&from=2026-01-01');
		expect(await screen.findByText(t('audit.empty.title'))).toBeInTheDocument();
		const clear = screen.getAllByRole('button', { name: t('audit.filters.clear') });
		expect(clear).toHaveLength(2);
		await user().click(clear[1]);
		expect(goto).toHaveBeenLastCalledWith('/admin/audit', { reset: false });
	});
});

describe('AuditScreen filters', () => {
	test('the URL drives the request', async () => {
		const api = mockApi({
			...baseRoutes,
			'GET /api/admin/audit': auditRoute([makeAuditEntry()], 60)
		});
		render('/admin/audit?event=user.*&success=true&actorId=2&from=2026-01-01&to=2026-01-31&page=2');
		await screen.findByText(t('audit.events.auth.login.success'));
		const query = api.calls('GET /api/admin/audit')[0].url.searchParams;
		expect(query.get('event')).toBe('user.*');
		expect(query.get('success')).toBe('true');
		expect(query.get('actorId')).toBe('2');
		expect(query.get('offset')).toBe('50');
		expect(query.get('from')).toMatch(/Z$/);
		expect(screen.getByLabelText(t('audit.filters.from'))).toHaveValue('2026-01-01');
		expect(screen.getByLabelText(t('audit.filters.event'))).toHaveTextContent(
			t('audit.filters.groups.user')
		);
		await vi.waitFor(() =>
			expect(screen.getByLabelText(t('audit.filters.actor'))).toHaveTextContent('peter')
		);
	});

	test('event, result and actor choices go to the URL from page 1', async () => {
		mockApi({ ...baseRoutes, 'GET /api/admin/audit': auditRoute([makeAuditEntry()], 120) });
		render('/admin/audit?page=2');
		await screen.findByText(t('audit.events.auth.login.success'));
		const u = user();
		await u.click(screen.getByLabelText(t('audit.filters.event')));
		await u.click(await findFloating('option', t('audit.filters.groups.privacy')));
		expect(goto).toHaveBeenLastCalledWith('/admin/audit?event=privacy.*', { reset: false });
		await u.click(screen.getByLabelText(t('audit.filters.event')));
		await u.click(await findFloating('option', t('audit.events.link.shared')));
		expect(goto).toHaveBeenLastCalledWith('/admin/audit?event=link.shared', { reset: false });
		await u.click(screen.getByLabelText(t('audit.filters.result')));
		await u.click(await findFloating('option', t('audit.result.failure')));
		expect(goto).toHaveBeenLastCalledWith('/admin/audit?event=link.shared&success=false', {
			reset: false
		});
		await u.click(screen.getByLabelText(t('audit.filters.actor')));
		await u.click(await findFloating('option', 'kanna', { exact: false }));
		expect(goto).toHaveBeenLastCalledWith(
			'/admin/audit?event=link.shared&success=false&actorId=1',
			{ reset: false }
		);
		await u.click(screen.getByLabelText(t('audit.filters.result')));
		await u.click(await findFloating('option', t('audit.filters.all')));
		await u.click(screen.getByLabelText(t('audit.filters.event')));
		await u.click(await findFloating('option', t('audit.filters.all')));
		expect(goto).toHaveBeenLastCalledWith('/admin/audit?actorId=1', { reset: false });
		await u.click(screen.getByRole('button', { name: t('audit.filters.clear') }));
		expect(goto).toHaveBeenLastCalledWith('/admin/audit', { reset: false });
	});

	test('dates are taken only when valid', async () => {
		mockApi({ ...baseRoutes, 'GET /api/admin/audit': auditRoute([makeAuditEntry()]) });
		render();
		await screen.findByText(t('audit.events.auth.login.success'));
		const from = screen.getByLabelText(t('audit.filters.from'));
		await fireEvent.change(from, { target: { value: '2026-03-01' } });
		expect(goto).toHaveBeenLastCalledWith('/admin/audit?from=2026-03-01', { reset: false });
		const to = screen.getByLabelText(t('audit.filters.to'));
		expect(to).toHaveAttribute('min', '2026-03-01');
		await fireEvent.change(to, { target: { value: '2026-03-31' } });
		expect(goto).toHaveBeenLastCalledWith('/admin/audit?from=2026-03-01&to=2026-03-31', {
			reset: false
		});
		expect(from).toHaveAttribute('max', '2026-03-31');
		const calls = vi.mocked(goto).mock.calls.length;
		await fireEvent.change(to, { target: { value: '' } });
		expect(goto).toHaveBeenLastCalledWith('/admin/audit?from=2026-03-01', { reset: false });
		expect(vi.mocked(goto).mock.calls.length).toBe(calls + 1);
	});

	test('the pager moves through the log', async () => {
		mockApi({ ...baseRoutes, 'GET /api/admin/audit': auditRoute([makeAuditEntry()], 75) });
		render();
		await screen.findByText(t('audit.events.auth.login.success'));
		await user().click(screen.getByRole('button', { name: t('admin.pager.next') }));
		expect(goto).toHaveBeenLastCalledWith('/admin/audit?page=2', { reset: false });
	});

	test('a failed user list leaves the actor filter with a note', async () => {
		mockApi({
			'GET /api/admin/settings': baseRoutes['GET /api/admin/settings'],
			'GET /api/admin/users': apiError(500, 'internal_error'),
			'GET /api/admin/audit': auditRoute([makeAuditEntry()])
		});
		render();
		await screen.findByText(t('audit.events.auth.login.success'));
		await user().click(screen.getByLabelText(t('audit.filters.actor')));
		await vi.waitFor(() =>
			expect(screen.getByText(t('audit.filters.actorUnavailable'))).toBeInTheDocument()
		);
	});

	test('a stale answer does not overwrite a newer one', async () => {
		let first = true;
		let releaseFirst: () => void = () => undefined;
		mockApi({
			...baseRoutes,
			'GET /api/admin/audit': async () => {
				if (first) {
					first = false;
					await new Promise<void>((resolve) => (releaseFirst = resolve));
					return json(page([makeAuditEntry({ event: 'auth.logout' })]));
				}
				return json(page([makeAuditEntry({ event: 'user.created' })]));
			}
		});
		render();
		setTestPage('/admin/audit?event=user.*');
		expect(await screen.findByText(t('audit.events.user.created'))).toBeInTheDocument();
		releaseFirst();
		await new Promise((resolve) => setTimeout(resolve, 20));
		expect(screen.queryByText(t('audit.events.auth.logout'))).toBeNull();
	});
});
