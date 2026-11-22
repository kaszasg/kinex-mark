// Sessions card of Settings > Security (plan 8.7.6).
import { screen, waitFor, within } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { toast } from 'svelte-sonner';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { goto } from '$app/navigation';
import { ApiError } from '$lib/api/errors.js';
import type { Session } from '$lib/api/types.js';
import { errorMessage, fmt, t } from '$lib/i18n/index.js';
import { session } from '$lib/state/session.svelte.js';
import { makeMe } from '$lib/test/fixtures.js';
import { apiError, json, mockApi, noContent } from '$lib/test/mock-api.js';
import { renderWithApp } from '$lib/test/render.js';
import SessionsCard, { orderSessions, sessionDevice } from './SessionsCard.svelte';

const user = () => userEvent.setup({ pointerEventsCheck: 0 });

const CHROME =
	'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0 Safari/537.36';

function makeSession(partial: Partial<Session> = {}): Session {
	return {
		id: 1,
		createdAt: '2026-11-20T08:00:00.000Z',
		lastSeenAt: '2026-11-21T09:30:00.000Z',
		expiresAt: '2026-11-27T08:00:00.000Z',
		ip: '192.0.2.10',
		userAgent: CHROME,
		current: false,
		...partial
	};
}

const SESSIONS: Session[] = [
	makeSession({ id: 7, userAgent: 'curl/8.4.0', ip: '2001:db8::1' }),
	makeSession({ id: 3, current: true }),
	makeSession({ id: 2, userAgent: '', ip: '' })
];

/** toHaveTextContent collapses the element's whitespace (NBSP included), so the expectation must too. */
function spaced(text: string): string {
	return text.replace(/\s+/g, ' ');
}

function list(): HTMLElement {
	return screen.getByRole('list', { name: t('security.sessions.listLabel') });
}

beforeEach(() => {
	vi.mocked(goto).mockClear();
	session.clear();
});

afterEach(() => {
	vi.unstubAllGlobals();
});

describe('session helpers', () => {
	test('device text: described, raw mono, unknown', () => {
		expect(sessionDevice({ userAgent: CHROME })).toEqual({ text: 'Chrome, Windows', mono: false });
		expect(sessionDevice({ userAgent: 'curl/8.4.0' })).toEqual({ text: 'curl/8.4.0', mono: true });
		expect(sessionDevice({ userAgent: '   ' })).toEqual({ text: null, mono: false });
	});

	test('the current session comes first', () => {
		expect(orderSessions(SESSIONS).map((s) => s.id)).toEqual([3, 7, 2]);
	});
});

describe('SessionsCard', () => {
	test('lists the sessions with device, IP and times; the current one is marked', async () => {
		mockApi({ 'GET /api/me/sessions': json({ items: SESSIONS }) });
		renderWithApp(SessionsCard, {}, { me: makeMe() });
		const items = within(
			await screen.findByRole('list', { name: t('security.sessions.listLabel') })
		).getAllByRole('listitem');
		expect(items).toHaveLength(3);

		const [current, curl, unknown] = items;
		expect(current).toHaveAttribute('data-current', 'true');
		expect(within(current).getByText('Chrome, Windows')).toHaveClass('font-medium');
		expect(within(current).getByText(t('security.sessions.current'))).toBeInTheDocument();
		expect(within(current).getByText('192.0.2.10')).toHaveClass('font-mono');
		expect(current).toHaveTextContent(spaced(fmt.dateTime('2026-11-20T08:00:00.000Z')));
		expect(current).toHaveTextContent(spaced(fmt.dateTime('2026-11-27T08:00:00.000Z')));
		expect(current.querySelector('time')).toHaveAttribute('datetime');

		expect(within(curl).getByText('curl/8.4.0')).toHaveClass('font-mono');
		expect(within(curl).queryByText(t('security.sessions.current'))).not.toBeInTheDocument();
		expect(within(unknown).getByText(t('security.sessions.unknownDevice'))).toBeInTheDocument();
		expect(within(unknown).getByText(fmt.missing)).toBeInTheDocument();
		expect(
			within(unknown).getByRole('button', {
				name: t('security.sessions.revoke', { device: t('security.sessions.unknownDevice') })
			})
		).toBeInTheDocument();
	});

	test('ending another session removes it at once', async () => {
		const api = mockApi({
			'GET /api/me/sessions': json({ items: SESSIONS }),
			'DELETE /api/me/sessions/7': noContent()
		});
		const success = vi.spyOn(toast, 'success');
		renderWithApp(SessionsCard, {}, { me: makeMe() });
		await screen.findByRole('list', { name: t('security.sessions.listLabel') });
		await user().click(
			screen.getByRole('button', {
				name: t('security.sessions.revoke', { device: 'curl/8.4.0' })
			})
		);
		await waitFor(() => expect(within(list()).getAllByRole('listitem')).toHaveLength(2));
		expect(api.calls('DELETE /api/me/sessions/7')).toHaveLength(1);
		expect(success).toHaveBeenCalledWith(t('security.sessions.revoked', { device: 'curl/8.4.0' }));
		expect(screen.queryByText('curl/8.4.0')).not.toBeInTheDocument();
	});

	test('a session that is already gone disappears with an error toast', async () => {
		mockApi({
			'GET /api/me/sessions': json({ items: SESSIONS }),
			'DELETE /api/me/sessions/7': apiError(404, 'not_found')
		});
		const failure = vi.spyOn(toast, 'error');
		renderWithApp(SessionsCard, {}, { me: makeMe() });
		await screen.findByRole('list', { name: t('security.sessions.listLabel') });
		await user().click(
			screen.getByRole('button', { name: t('security.sessions.revoke', { device: 'curl/8.4.0' }) })
		);
		await waitFor(() => expect(within(list()).getAllByRole('listitem')).toHaveLength(2));
		expect(failure).toHaveBeenCalledWith(errorMessage(new ApiError(404, 'not_found')));
	});

	test('a failed revoke keeps the row', async () => {
		mockApi({
			'GET /api/me/sessions': json({ items: SESSIONS }),
			'DELETE /api/me/sessions/7': apiError(500, 'internal_error')
		});
		const failure = vi.spyOn(toast, 'error');
		renderWithApp(SessionsCard, {}, { me: makeMe() });
		await screen.findByRole('list', { name: t('security.sessions.listLabel') });
		await user().click(
			screen.getByRole('button', { name: t('security.sessions.revoke', { device: 'curl/8.4.0' }) })
		);
		await waitFor(() => expect(failure).toHaveBeenCalledOnce());
		expect(within(list()).getAllByRole('listitem')).toHaveLength(3);
	});

	test('ending the current session asks first, then goes to the login page', async () => {
		const api = mockApi({
			'GET /api/me/sessions': json({ items: SESSIONS }),
			'DELETE /api/me/sessions/3': noContent()
		});
		renderWithApp(SessionsCard, {}, { me: makeMe() });
		await screen.findByRole('list', { name: t('security.sessions.listLabel') });
		const button = screen.getByRole('button', {
			name: t('security.sessions.revoke', { device: 'Chrome, Windows' })
		});

		await user().click(button);
		const dialog = await screen.findByRole('alertdialog');
		expect(dialog).toHaveTextContent(t('security.sessions.revokeCurrent.title'));
		await user().click(within(dialog).getByRole('button', { name: t('common.actions.cancel') }));
		await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument());
		expect(api.calls('DELETE /api/me/sessions/3')).toHaveLength(0);

		await user().click(button);
		await user().click(
			await screen.findByRole('button', { name: t('security.sessions.revokeCurrent.submit') })
		);
		await waitFor(() => expect(goto).toHaveBeenCalledWith('/login'));
		expect(api.calls('DELETE /api/me/sessions/3')).toHaveLength(1);
		expect(session.status).toBe('anonymous');
		expect(session.lastClearReason).toBe('logout');
	});

	test('signs out every other device with a counted toast', async () => {
		const api = mockApi({
			'GET /api/me/sessions': json({ items: SESSIONS }),
			'POST /api/me/sessions/revoke-others': json({ revoked: 2 })
		});
		const success = vi.spyOn(toast, 'success');
		renderWithApp(SessionsCard, {}, { me: makeMe() });
		await screen.findByRole('list', { name: t('security.sessions.listLabel') });
		const others = screen.getByRole('button', { name: t('security.sessions.revokeOthers') });
		expect(others).toBeEnabled();
		await user().click(others);
		await waitFor(() => expect(within(list()).getAllByRole('listitem')).toHaveLength(1));
		expect(api.calls('POST /api/me/sessions/revoke-others')).toHaveLength(1);
		expect(success).toHaveBeenCalledWith(t('security.sessions.revokedOthers', { count: 2 }));
		expect(others).toBeDisabled();
	});

	test('a failed sign-out of other devices keeps the list', async () => {
		mockApi({
			'GET /api/me/sessions': json({ items: SESSIONS }),
			'POST /api/me/sessions/revoke-others': apiError(500, 'internal_error')
		});
		const failure = vi.spyOn(toast, 'error');
		renderWithApp(SessionsCard, {}, { me: makeMe() });
		await screen.findByRole('list', { name: t('security.sessions.listLabel') });
		await user().click(screen.getByRole('button', { name: t('security.sessions.revokeOthers') }));
		await waitFor(() =>
			expect(failure).toHaveBeenCalledWith(errorMessage(new ApiError(500, 'internal_error')))
		);
		expect(within(list()).getAllByRole('listitem')).toHaveLength(3);
	});

	test('a network failure shows an alert with retry', async () => {
		const api = mockApi({});
		api.fetch.mockRejectedValueOnce(new TypeError('Failed to fetch'));
		renderWithApp(SessionsCard, {}, { me: makeMe() });
		const alert = await screen.findByRole('alert');
		expect(alert).toHaveTextContent(t('errors.network_error.title'));
		expect(
			screen.getByRole('button', { name: t('security.sessions.revokeOthers') })
		).toBeDisabled();

		api.route('GET /api/me/sessions', json({ items: [makeSession({ id: 3, current: true })] }));
		await user().click(within(alert).getByRole('button', { name: t('common.actions.retry') }));
		await screen.findByRole('list', { name: t('security.sessions.listLabel') });
		expect(
			screen.getByRole('button', { name: t('security.sessions.revokeOthers') })
		).toBeDisabled();
	});

	test('an empty list shows the empty state', async () => {
		mockApi({ 'GET /api/me/sessions': json({ items: [] }) });
		renderWithApp(SessionsCard, {}, { me: makeMe() });
		expect(await screen.findByText(t('security.sessions.empty.title'))).toBeInTheDocument();
		expect(screen.getByText(t('security.sessions.empty.text'))).toBeInTheDocument();
	});

	test('the loading rows appear while the list is slow', async () => {
		let release: () => void = () => undefined;
		mockApi({
			'GET /api/me/sessions': () =>
				new Promise((resolve) => {
					release = () => resolve(json({ items: SESSIONS }));
				})
		});
		renderWithApp(SessionsCard, {}, { me: makeMe() });
		expect(await screen.findAllByTestId('loading-row')).toHaveLength(2);
		release();
		await screen.findByRole('list', { name: t('security.sessions.listLabel') });
		expect(screen.queryByTestId('loading-row')).not.toBeInTheDocument();
	});
});
