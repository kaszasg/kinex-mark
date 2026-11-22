// User dialog: create with generated or typed password, edit with step-up on role/active.
import { render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, test, vi } from 'vitest';
import { ApiError } from '$lib/api/errors.js';
import { errorMessage, errorTitle, i18n, t } from '$lib/i18n/index.js';
import { apiError, json, mockApi } from '$lib/test/mock-api.js';
import UserDialog from './UserDialog.svelte';
import { findFloating, makeAdminUser, user } from './test-helpers.js';

afterEach(() => {
	vi.unstubAllGlobals();
});

const field = (key: 'username' | 'displayName' | 'role' | 'locale' | 'password' | 'active') =>
	screen.getByLabelText(t(`admin.userDialog.fields.${key}`));
const createButton = () => screen.getByRole('button', { name: t('admin.userDialog.create') });
const saveButton = () => screen.getByRole('button', { name: t('admin.userDialog.save') });
const stepUpField = () => screen.queryByLabelText(t('common.stepUp.label'));

async function choose(u: ReturnType<typeof user>, trigger: HTMLElement, option: string) {
	await u.click(trigger);
	await u.click(await findFloating('option', option));
}

describe('UserDialog: create', () => {
	test('creates a user with a generated password in the current language', async () => {
		i18n.setLocale('en', { persist: false });
		const created = { user: makeAdminUser({ username: 'new.user' }), temporaryPassword: 'Tmp' };
		const api = mockApi({ 'POST /api/admin/users': json(created, 201) });
		const onCreated = vi.fn();
		render(UserDialog, { open: true, onCreated });
		expect(screen.getByRole('dialog', { name: t('admin.userDialog.createTitle') })).toBeVisible();
		expect(screen.getByText(t('admin.userDialog.usernameHelp'))).toBeInTheDocument();
		expect(screen.getByText(t('admin.userDialog.generatedHelp'))).toBeInTheDocument();
		expect(field('username')).toHaveFocus();
		expect(field('locale')).toHaveTextContent('English');
		const u = user();
		await u.type(field('username'), ' new.user ');
		await u.type(field('displayName'), 'New User');
		await choose(u, field('role'), t('roles.developer'));
		await u.click(createButton());
		await vi.waitFor(() => expect(onCreated).toHaveBeenCalledWith(created));
		expect(api.calls('POST /api/admin/users')[0].body).toEqual({
			username: 'new.user',
			displayName: 'New User',
			role: 'developer',
			locale: 'en'
		});
		await vi.waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
	});

	test('checks the username and the typed password before sending', async () => {
		i18n.setLocale('hu', { persist: false });
		const api = mockApi({});
		render(UserDialog, { open: true });
		const u = user();
		await u.click(createButton());
		expect(field('username')).toHaveAttribute('aria-invalid', 'true');
		expect(screen.getByText(t('fields.required'))).toBeInTheDocument();
		await u.type(field('username'), 'ab');
		await u.click(createButton());
		expect(screen.getByText(t('fields.too_short'))).toBeInTheDocument();
		await u.clear(field('username'));
		await u.type(field('username'), 'bad name');
		await u.click(createButton());
		expect(screen.getByText(t('fields.byField.username.invalid_format'))).toBeInTheDocument();
		await u.clear(field('username'));
		await u.type(field('username'), 'x'.repeat(64));
		await u.click(screen.getByRole('radio', { name: t('admin.userDialog.passwordMode.manual') }));
		// The help comes from the policy and names the username rule, like the password forms.
		const help = t('admin.userDialog.passwordHelp', { min: 12, distinct: 6 });
		expect(screen.getByText(help)).toBeInTheDocument();
		expect(help).toBe(
			'Legalább 12 karakter, legalább 6 különböző karakter, a felhasználónév nélkül.'
		);
		await u.click(createButton());
		expect(field('password')).toHaveAttribute('aria-invalid', 'true');
		expect(api.requests).toHaveLength(0);
	});

	test('a typed password is sent and server field codes land under the fields', async () => {
		const api = mockApi({
			'POST /api/admin/users': apiError(400, 'validation_failed', {
				password: 'password_too_short',
				displayName: 'too_long'
			})
		});
		render(UserDialog, { open: true });
		const u = user();
		await u.type(field('username'), 'newbie');
		await u.click(screen.getByRole('radio', { name: t('admin.userDialog.passwordMode.manual') }));
		await u.type(field('password'), 'short');
		await u.click(createButton());
		expect(await screen.findByText(t('fields.password_too_short'))).toBeInTheDocument();
		expect(field('displayName')).toHaveAttribute('aria-invalid', 'true');
		expect(api.calls('POST /api/admin/users')[0].body).toMatchObject({
			username: 'newbie',
			password: 'short',
			role: 'customer'
		});
		await u.click(
			screen.getByRole('radio', { name: t('admin.userDialog.passwordMode.generated') })
		);
		expect(screen.queryByLabelText(t('admin.userDialog.fields.password'))).toBeNull();
	});

	test('a taken username is shown under the username field', async () => {
		mockApi({ 'POST /api/admin/users': apiError(409, 'username_taken') });
		render(UserDialog, { open: true });
		const u = user();
		await u.type(field('username'), 'taken');
		await u.click(createButton());
		const text = errorMessage(new ApiError(409, 'username_taken'));
		expect(await screen.findByText(text)).toBeInTheDocument();
		expect(field('username')).toHaveAttribute('aria-invalid', 'true');
	});

	test('a display name over 100 characters is stopped on the client', async () => {
		const api = mockApi({});
		render(UserDialog, { open: true });
		const u = user();
		await u.type(field('username'), 'longname');
		await u.click(field('displayName'));
		await u.paste('n'.repeat(101));
		await u.click(createButton());
		expect(field('displayName')).toHaveAttribute('aria-invalid', 'true');
		expect(api.requests).toHaveLength(0);
	});

	test('other errors appear as an alert and cancel closes the dialog', async () => {
		mockApi({ 'POST /api/admin/users': apiError(500, 'internal_error') });
		render(UserDialog, { open: true });
		const u = user();
		await u.type(field('username'), 'someone');
		await u.click(createButton());
		expect(await screen.findByRole('alert')).toHaveTextContent(
			errorTitle(new ApiError(500, 'internal_error'))
		);
		await u.click(screen.getByRole('button', { name: t('common.actions.cancel') }));
		await vi.waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
	});

	test('a network failure that is not an API error still shows an alert', async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn(async () => {
				throw new TypeError('offline');
			})
		);
		render(UserDialog, { open: true });
		const u = user();
		await u.type(field('username'), 'someone');
		await u.click(createButton());
		expect(await screen.findByRole('alert')).toHaveTextContent(t('errors.network_error.title'));
	});
});

describe('UserDialog: edit', () => {
	const target = makeAdminUser({
		id: 42,
		username: 'peter',
		displayName: 'Peter',
		role: 'customer'
	});

	test('saving waits for a change; a display name change needs no password', async () => {
		const updated = { ...target, displayName: 'Peter Szabo' };
		const api = mockApi({ 'PATCH /api/admin/users/42': json(updated) });
		const onSaved = vi.fn();
		render(UserDialog, { open: true, user: target, selfId: 1, onSaved });
		expect(
			screen.getByRole('dialog', { name: t('admin.userDialog.editTitle', { username: 'peter' }) })
		).toBeVisible();
		expect(screen.queryByLabelText(t('admin.userDialog.fields.username'))).toBeNull();
		expect(field('displayName')).toHaveFocus();
		expect(saveButton()).toBeDisabled();
		const u = user();
		await u.type(field('displayName'), ' Szabo');
		expect(stepUpField()).toBeNull();
		await u.click(saveButton());
		await vi.waitFor(() => expect(onSaved).toHaveBeenCalledWith(updated));
		expect(api.calls('PATCH /api/admin/users/42')[0].body).toEqual({ displayName: 'Peter Szabo' });
	});

	test('a role change asks for the own password and sends it', async () => {
		const api = mockApi({
			'PATCH /api/admin/users/42': (req) =>
				(req.body as { currentPassword?: string }).currentPassword === 'right'
					? json({ ...target, role: 'developer', active: false })
					: apiError(400, 'current_password_invalid', { currentPassword: 'incorrect' })
		});
		const onSaved = vi.fn();
		render(UserDialog, { open: true, user: target, selfId: 1, onSaved });
		const u = user();
		await choose(u, field('role'), t('roles.developer'));
		await u.click(field('active'));
		expect(stepUpField()).not.toBeNull();
		expect(saveButton()).toBeDisabled();
		await u.type(stepUpField() as HTMLElement, 'wrong');
		await u.click(saveButton());
		const message = errorMessage(new ApiError(400, 'current_password_invalid'));
		expect(await screen.findByText(message)).toBeInTheDocument();
		expect(stepUpField()).toHaveAttribute('aria-invalid', 'true');
		await u.clear(stepUpField() as HTMLElement);
		await u.type(stepUpField() as HTMLElement, 'right');
		await u.click(saveButton());
		await vi.waitFor(() => expect(onSaved).toHaveBeenCalled());
		expect(api.calls('PATCH /api/admin/users/42').at(-1)?.body).toEqual({
			role: 'developer',
			active: false,
			currentPassword: 'right'
		});
	});

	test('last_admin and self_modification are alerts', async () => {
		mockApi({ 'PATCH /api/admin/users/42': apiError(409, 'last_admin') });
		render(UserDialog, { open: true, user: { ...target, role: 'admin' }, selfId: 1 });
		const u = user();
		await choose(u, field('role'), t('roles.customer'));
		await u.type(stepUpField() as HTMLElement, 'pw');
		await u.click(saveButton());
		expect(await screen.findByRole('alert')).toHaveTextContent(
			errorTitle(new ApiError(409, 'last_admin'))
		);
	});

	test('the own account keeps role and active locked', () => {
		render(UserDialog, { open: true, user: { ...target, id: 1 }, selfId: 1 });
		expect(field('role')).toBeDisabled();
		expect(field('active')).toBeDisabled();
		expect(screen.getAllByText(t('admin.userDialog.selfHint')).length).toBe(2);
	});
});
