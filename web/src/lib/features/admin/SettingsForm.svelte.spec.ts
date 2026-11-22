// The /admin/settings screen: loading, error, edit and save, validation, locks, jobs (plan 8.7.8).
import { render, screen, within } from '@testing-library/svelte';
import { toast } from 'svelte-sonner';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { beforeNavigate, goto } from '$app/navigation';
import { ApiError } from '$lib/api/errors.js';
import { errorMessage, errorTitle, fieldMessage, t } from '$lib/i18n/index.js';
import { apiError, json, mockApi } from '$lib/test/mock-api.js';
import SettingsForm from './SettingsForm.svelte';
import { makeSettingsResponse, user } from './test-helpers.js';
import { normalizeSpace } from '$lib/test/text.js';

vi.mock('svelte-sonner', async (importOriginal) => ({
	...(await importOriginal<typeof import('svelte-sonner')>()),
	toast: Object.assign(vi.fn(), { success: vi.fn(), error: vi.fn() })
}));

type Guard = Parameters<typeof beforeNavigate>[0];

const field = (key: string) => screen.getByLabelText(t(`admin.settings.fields.${key}` as never));
const saveButton = () => screen.getByRole('button', { name: t('admin.settings.save') });
const discardButton = () => screen.getByRole('button', { name: t('admin.settings.discard') });

async function loaded() {
	render(SettingsForm);
	await screen.findByRole('button', { name: t('admin.settings.save') });
}

function navigateAway(type: 'link' | 'leave' = 'link', to = 'http://localhost/links') {
	const guard = vi.mocked(beforeNavigate).mock.calls.at(-1)?.[0] as Guard;
	const cancel = vi.fn();
	guard({ type, to: { url: new URL(to) }, cancel } as unknown as Parameters<Guard>[0]);
	return cancel;
}

beforeEach(() => {
	vi.mocked(toast.success).mockClear();
	vi.mocked(toast.error).mockClear();
	vi.mocked(goto).mockReset();
	vi.mocked(goto).mockResolvedValue(undefined);
});

afterEach(() => {
	vi.unstubAllGlobals();
});

describe('SettingsForm states', () => {
	test('loading skeleton, then every card with the saved values', async () => {
		mockApi({ 'GET /api/admin/settings': json(makeSettingsResponse()) });
		render(SettingsForm);
		expect(screen.getByRole('heading', { name: t('admin.settings.pageTitle') })).toBeVisible();
		expect(screen.queryByRole('button', { name: t('admin.settings.save') })).toBeNull();
		await screen.findByRole('button', { name: t('admin.settings.save') });
		for (const card of ['archive', 'sessions', 'audit', 'fetch', 'privacy'] as const)
			expect(screen.getByText(t(`admin.settings.cards.${card}.title`))).toBeInTheDocument();
		expect(field('defaultArchiveDays')).toHaveValue(90);
		expect(field('sessionIdleMinutes')).toHaveValue(720);
		expect(field('privacyControllerEmail')).toHaveValue('privacy@example.com');
		expect(screen.getByTestId('notice-version')).toHaveTextContent('3');
		expect(screen.getByText(t('admin.settings.noChanges'))).toBeInTheDocument();
		expect(saveButton()).toBeDisabled();
		expect(discardButton()).toBeDisabled();
		expect(
			screen.getByText(
				normalizeSpace(t('admin.settings.help.days', { min: 30, max: 3650, default: 180 }))
			)
		).toBeInTheDocument();
	});

	test('a load error offers a retry', async () => {
		let fail = true;
		mockApi({
			'GET /api/admin/settings': () =>
				fail ? apiError(500, 'internal_error') : json(makeSettingsResponse())
		});
		render(SettingsForm);
		const alert = await screen.findByRole('alert');
		expect(alert).toHaveTextContent(errorTitle(new ApiError(500, 'internal_error')));
		fail = false;
		await user().click(within(alert).getByRole('button', { name: t('common.actions.retry') }));
		expect(await screen.findByRole('button', { name: t('admin.settings.save') })).toBeVisible();
	});
});

describe('SettingsForm editing', () => {
	test('changes enable saving; the patch holds only the changes', async () => {
		const api = mockApi({
			'GET /api/admin/settings': json(makeSettingsResponse()),
			'PATCH /api/admin/settings': json(
				makeSettingsResponse({ defaultArchiveDays: 30, allowPrivateFetch: true })
			)
		});
		await loaded();
		const u = user();
		await u.clear(field('defaultArchiveDays'));
		await u.type(field('defaultArchiveDays'), '30');
		await u.click(screen.getByRole('switch'));
		expect(screen.getByText(t('admin.settings.unsaved'))).toBeInTheDocument();
		await u.click(saveButton());
		await vi.waitFor(() => expect(toast.success).toHaveBeenCalledWith(t('admin.settings.saved')));
		expect(api.calls('PATCH /api/admin/settings')[0].body).toEqual({
			defaultArchiveDays: 30,
			allowPrivateFetch: true
		});
		expect(saveButton()).toBeDisabled();
		expect(field('defaultArchiveDays')).toHaveValue(30);
	});

	test('discard returns to the saved values', async () => {
		mockApi({ 'GET /api/admin/settings': json(makeSettingsResponse()) });
		await loaded();
		const u = user();
		await u.type(field('privacyDpoContact'), 'dpo');
		expect(discardButton()).toBeEnabled();
		await u.click(discardButton());
		expect(field('privacyDpoContact')).toHaveValue('');
		expect(saveButton()).toBeDisabled();
	});

	test('client validation marks the fields and sends nothing', async () => {
		const api = mockApi({ 'GET /api/admin/settings': json(makeSettingsResponse()) });
		await loaded();
		const u = user();
		await u.clear(field('sessionIdleMinutes'));
		await u.type(field('sessionIdleMinutes'), '5');
		await u.clear(field('privacyControllerEmail'));
		await u.type(field('privacyControllerEmail'), 'nope');
		await u.click(saveButton());
		expect(field('sessionIdleMinutes')).toHaveAttribute('aria-invalid', 'true');
		// Testing Library collapses the no-break space of the grouped number in the page text.
		const range = fieldMessage('sessionIdleMinutes', 'out_of_range', { min: 15, max: 10080 });
		expect(screen.getByText(range.replace(/\s+/g, ' '))).toBeInTheDocument();
		expect(field('privacyControllerEmail')).toHaveAttribute('aria-invalid', 'true');
		await vi.waitFor(() => expect(field('sessionIdleMinutes')).toHaveFocus());
		expect(api.calls('PATCH /api/admin/settings')).toHaveLength(0);
		await u.type(field('sessionIdleMinutes'), '0');
		expect(field('sessionIdleMinutes')).not.toHaveAttribute('aria-invalid');
	});

	test('server field codes land under the fields', async () => {
		mockApi({
			'GET /api/admin/settings': json(makeSettingsResponse()),
			'PATCH /api/admin/settings': apiError(400, 'validation_failed', {
				auditRetentionDays: 'out_of_range',
				privacyControllerName: 'too_long'
			})
		});
		await loaded();
		const u = user();
		await u.clear(field('auditRetentionDays'));
		await u.type(field('auditRetentionDays'), '400');
		await u.click(saveButton());
		await vi.waitFor(() =>
			expect(field('auditRetentionDays')).toHaveAttribute('aria-invalid', 'true')
		);
		expect(
			screen.getByText(fieldMessage('auditRetentionDays', 'out_of_range', { min: 30, max: 3650 }))
		).toBeInTheDocument();
		expect(field('privacyControllerName')).toHaveAttribute('aria-invalid', 'true');
	});

	test('setting_locked and other save errors appear as an alert', async () => {
		mockApi({
			'GET /api/admin/settings': json(makeSettingsResponse()),
			'PATCH /api/admin/settings': apiError(409, 'setting_locked')
		});
		await loaded();
		const u = user();
		await u.type(field('privacyDpoContact'), 'dpo');
		await u.click(saveButton());
		expect(await screen.findByRole('alert')).toHaveTextContent(
			errorTitle(new ApiError(409, 'setting_locked'))
		);
		expect(saveButton()).toBeEnabled();
	});

	test('a value locked by the environment is read-only with an explanation', async () => {
		mockApi({
			'GET /api/admin/settings': json(
				makeSettingsResponse({ allowPrivateFetch: true }, [
					'allowPrivateFetch',
					'sessionIdleMinutes'
				])
			)
		});
		await loaded();
		expect(screen.getByRole('switch')).toBeDisabled();
		expect(screen.getByRole('switch')).toBeChecked();
		expect(screen.getByText(t('admin.settings.locked.title'))).toBeInTheDocument();
		expect(screen.getByText(t('admin.settings.locked.privateFetch'))).toBeInTheDocument();
		expect(field('sessionIdleMinutes')).toBeDisabled();
		expect(
			screen.getByText(t('admin.settings.locked.generic'), { exact: false })
		).toBeInTheDocument();
	});
});

describe('SettingsForm jobs', () => {
	test('the archive job reports the archived count', async () => {
		mockApi({
			'GET /api/admin/settings': json(makeSettingsResponse()),
			'POST /api/admin/jobs/archive': json({ archived: 4 })
		});
		await loaded();
		await user().click(screen.getByRole('button', { name: t('admin.settings.runArchive') }));
		await vi.waitFor(() =>
			expect(toast.success).toHaveBeenCalledWith(t('admin.settings.archived', { count: 4 }))
		);
	});

	test('a failed archive job is an error toast', async () => {
		mockApi({
			'GET /api/admin/settings': json(makeSettingsResponse()),
			'POST /api/admin/jobs/archive': apiError(500, 'internal_error')
		});
		await loaded();
		await user().click(screen.getByRole('button', { name: t('admin.settings.runArchive') }));
		await vi.waitFor(() =>
			expect(toast.error).toHaveBeenCalledWith(errorMessage(new ApiError(500, 'internal_error')))
		);
	});

	test('an archive job without an API answer reports a network error', async () => {
		mockApi({ 'GET /api/admin/settings': json(makeSettingsResponse()) });
		await loaded();
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => {
				throw new TypeError('offline');
			})
		);
		await user().click(screen.getByRole('button', { name: t('admin.settings.runArchive') }));
		await vi.waitFor(() =>
			expect(toast.error).toHaveBeenCalledWith(errorMessage(new ApiError(0, 'network_error')))
		);
	});

	test('the notice bump is confirmed and keeps unsaved edits', async () => {
		const api = mockApi({
			'GET /api/admin/settings': json(makeSettingsResponse()),
			'POST /api/admin/settings/privacy-notice/bump': json(
				makeSettingsResponse({
					privacyNoticeVersion: 4,
					privacyNoticeUpdatedAt: '2026-02-01T08:00:00Z'
				})
			)
		});
		await loaded();
		const u = user();
		await u.type(field('privacyDpoContact'), 'dpo');
		await u.click(screen.getByRole('button', { name: t('admin.settings.notice.bump') }));
		const dialog = await screen.findByRole('alertdialog', {
			name: t('admin.settings.notice.bumpTitle')
		});
		expect(within(dialog).getByText(t('admin.settings.notice.bumpText'))).toBeVisible();
		await u.click(within(dialog).getByRole('button', { name: t('admin.settings.notice.bump') }));
		await vi.waitFor(() =>
			expect(toast.success).toHaveBeenCalledWith(t('admin.settings.notice.bumped', { version: 4 }))
		);
		expect(screen.getByTestId('notice-version')).toHaveTextContent('4');
		expect(field('privacyDpoContact')).toHaveValue('dpo');
		expect(saveButton()).toBeEnabled();
		expect(api.calls('POST /api/admin/settings/privacy-notice/bump')).toHaveLength(1);
	});
});

describe('SettingsForm leave guard', () => {
	test('without changes navigation is not stopped', async () => {
		mockApi({ 'GET /api/admin/settings': json(makeSettingsResponse()) });
		await loaded();
		expect(navigateAway()).not.toHaveBeenCalled();
	});

	test('with changes the user confirms before leaving', async () => {
		mockApi({ 'GET /api/admin/settings': json(makeSettingsResponse()) });
		await loaded();
		const u = user();
		await u.type(field('privacyDpoContact'), 'dpo');
		const cancel = navigateAway();
		expect(cancel).toHaveBeenCalled();
		const dialog = await screen.findByRole('alertdialog', {
			name: t('admin.settings.leave.title')
		});
		await u.click(within(dialog).getByRole('button', { name: t('admin.settings.leave.action') }));
		await vi.waitFor(() => expect(goto).toHaveBeenCalledWith(new URL('http://localhost/links')));
		expect(navigateAway()).not.toHaveBeenCalled();
	});

	test('closing the tab is left to the browser', async () => {
		mockApi({ 'GET /api/admin/settings': json(makeSettingsResponse()) });
		await loaded();
		await user().type(field('privacyDpoContact'), 'dpo');
		expect(navigateAway('leave')).toHaveBeenCalled();
		expect(screen.queryByRole('alertdialog')).toBeNull();
	});

	test('staying keeps the edits', async () => {
		mockApi({ 'GET /api/admin/settings': json(makeSettingsResponse()) });
		await loaded();
		const u = user();
		await u.type(field('privacyDpoContact'), 'dpo');
		navigateAway();
		const dialog = await screen.findByRole('alertdialog');
		await u.click(within(dialog).getByRole('button', { name: t('common.actions.cancel') }));
		await vi.waitFor(() => expect(screen.queryByRole('alertdialog')).toBeNull());
		expect(goto).not.toHaveBeenCalled();
		expect(field('privacyDpoContact')).toHaveValue('dpo');
	});
});
