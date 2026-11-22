// Error mapping of the security and privacy forms (plan 8.7.6).
import { describe, expect, test } from 'vitest';
import { ApiError } from '$lib/api/errors.js';
import { errorMessage, fieldMessage } from '$lib/i18n/index.js';
import { focusFirstInvalid, formErrors } from './form-errors.js';

describe('formErrors', () => {
	test('a non-API error goes to the alert', () => {
		const err = new Error('boom');
		expect(formErrors(err, {}, [])).toEqual({ fields: {}, alert: err });
	});

	test('a mapped code goes under its field as one sentence', () => {
		const err = new ApiError(400, 'current_password_invalid');
		expect(formErrors(err, { current_password_invalid: 'password' }, ['password'])).toEqual({
			fields: { password: errorMessage(err) },
			alert: null
		});
	});

	test('validation codes of known fields go under the fields', () => {
		const err = new ApiError(400, 'validation_failed', { code: 'required' });
		expect(formErrors(err, {}, ['code'])).toEqual({
			fields: { code: fieldMessage('code', 'required') },
			alert: null
		});
	});

	test('unknown validation fields keep the alert', () => {
		const err = new ApiError(400, 'validation_failed', { code: 'required', other: 'invalid' });
		const result = formErrors(err, {}, ['code']);
		expect(result.fields).toEqual({ code: fieldMessage('code', 'required') });
		expect(result.alert).toBe(err);
		expect(formErrors(new ApiError(400, 'validation_failed'), {}, ['code'])).toEqual({
			fields: {},
			alert: expect.any(ApiError)
		});
	});

	test('other codes go to the alert', () => {
		const err = new ApiError(409, 'last_admin');
		expect(formErrors(err, { invalid_totp: 'code' }, ['code'])).toEqual({ fields: {}, alert: err });
	});
});

describe('focusFirstInvalid', () => {
	test('focuses the first invalid control and tolerates a missing root', async () => {
		const root = document.createElement('div');
		root.innerHTML =
			'<input id="a" /><input id="b" aria-invalid="true" /><input id="c" aria-invalid="true" />';
		document.body.append(root);
		await focusFirstInvalid(root);
		expect(document.activeElement?.id).toBe('b');
		root.remove();
		await expect(focusFirstInvalid(null)).resolves.toBeUndefined();
	});
});
