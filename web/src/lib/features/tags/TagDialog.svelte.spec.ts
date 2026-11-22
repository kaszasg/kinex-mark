// Tag dialog of plan 8.7.5: validation, colour choice, preview and every error path.
import { render, screen, within } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { toast } from 'svelte-sonner';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { ApiError } from '$lib/api/errors.js';
import { errorMessage, t } from '$lib/i18n/index.js';
import { TagsStore } from '$lib/state/tags.svelte.js';
import { makeTag } from '$lib/test/fixtures.js';
import TagDialog, { TAG_NAME_MAX } from './TagDialog.svelte';

const user = () => userEvent.setup({ pointerEventsCheck: 0 });

function open(props: { tag?: ReturnType<typeof makeTag> | null; onSaved?: () => void } = {}) {
	const store = new TagsStore();
	render(TagDialog, { open: true, store, ...props });
	const dialog = screen.getByRole('dialog');
	const name = within(dialog).getByLabelText(t('tags.dialog.name'));
	const save = within(dialog).getByRole('button', { name: t('tags.dialog.save') });
	return { store, dialog, name, save };
}

beforeEach(() => {
	vi.spyOn(toast, 'success').mockImplementation(() => 1);
});

afterEach(() => {
	vi.unstubAllGlobals();
});

describe('TagDialog', () => {
	test('lists the seven colours in palette order, neutral first chosen, and previews the chip', async () => {
		const { dialog, name } = open();
		const radios = within(dialog).getAllByRole('radio');
		expect(radios.map((radio) => radio.textContent?.trim())).toEqual(
			['blue', 'ochre', 'green', 'violet', 'rose', 'neutral', 'contrast'].map((c) =>
				t(`tagColors.${c as 'blue'}`)
			)
		);
		expect(within(dialog).getByRole('radio', { name: t('tagColors.neutral') })).toHaveAttribute(
			'aria-checked',
			'true'
		);
		const preview = within(dialog).getByText(t('tags.dialog.preview')).nextElementSibling;
		expect(preview).toHaveTextContent(t('tagColors.neutral'));
		await user().type(name, 'docs');
		expect(preview).toHaveTextContent('docs');
		expect(within(dialog).getByTestId('tag-name-counter')).toHaveTextContent(`4/${TAG_NAME_MAX}`);
		await user().click(within(dialog).getByRole('radio', { name: t('tagColors.rose') }));
		expect(preview?.querySelector('.bg-chart-5')).not.toBeNull();
	});

	test('clicking the chosen colour again keeps it', async () => {
		const { dialog } = open();
		const neutral = within(dialog).getByRole('radio', { name: t('tagColors.neutral') });
		await user().click(neutral);
		expect(neutral).toHaveAttribute('aria-checked', 'true');
	});

	test('an empty name is required and sends nothing', async () => {
		const { store, name, save } = open();
		const create = vi.spyOn(store, 'create');
		await user().type(name, '   ');
		await user().click(save);
		expect(create).not.toHaveBeenCalled();
		expect(name).toHaveAttribute('aria-invalid', 'true');
		expect(screen.getByText(t('fields.required'))).toBeInTheDocument();
		await user().type(name, 'x');
		expect(name).not.toHaveAttribute('aria-invalid');
	});

	test('a name over 40 characters is too long', async () => {
		const { store, dialog, name, save } = open();
		const create = vi.spyOn(store, 'create');
		await user().type(name, 'a'.repeat(TAG_NAME_MAX + 1));
		expect(within(dialog).getByTestId('tag-name-counter')).toHaveClass('text-destructive');
		await user().click(save);
		expect(create).not.toHaveBeenCalled();
		expect(screen.getByText(t('fields.too_long'))).toBeInTheDocument();
	});

	test('a duplicate name is reported under the name field', async () => {
		const { store, name, save } = open();
		const duplicate = new ApiError(409, 'tag_duplicate', {}, { tagId: 4 });
		vi.spyOn(store, 'create').mockRejectedValue(duplicate);
		await user().type(name, 'infra');
		await user().click(save);
		expect(await screen.findByText(errorMessage(duplicate))).toBeInTheDocument();
		expect(name).toHaveAttribute('aria-invalid', 'true');
		expect(screen.getByRole('dialog')).toBeInTheDocument();
		expect(screen.queryByRole('alert')).toBeNull();
	});

	test('the tag limit is an alert', async () => {
		const { store, name, save } = open();
		vi.spyOn(store, 'create').mockRejectedValue(new ApiError(409, 'tag_limit_reached'));
		await user().type(name, 'infra');
		await user().click(save);
		expect(await screen.findByRole('alert')).toHaveTextContent(t('errors.tag_limit_reached.title'));
	});

	test('a server-side field error lands on the name field', async () => {
		const { store, name, save } = open();
		vi.spyOn(store, 'create').mockRejectedValue(
			new ApiError(400, 'validation_failed', { name: 'invalid' })
		);
		await user().type(name, 'infra');
		await user().click(save);
		expect(await screen.findByText(t('fields.invalid'))).toBeInTheDocument();
		expect(screen.queryByRole('alert')).toBeNull();
	});

	test('a failure without a code reads as a network error', async () => {
		const { store, name, save } = open();
		vi.spyOn(store, 'create').mockRejectedValue(new TypeError('offline'));
		await user().type(name, 'infra');
		await user().click(save);
		expect(await screen.findByRole('alert')).toHaveTextContent(t('errors.network_error.title'));
	});

	test('saving shows the spinner and blocks closing until the answer', async () => {
		const { store, name, save } = open();
		let resolve: (tag: ReturnType<typeof makeTag>) => void = () => {};
		vi.spyOn(store, 'create').mockImplementation(() => new Promise((r) => (resolve = r)));
		await user().type(name, 'ops');
		await user().click(save);
		expect(save).toBeDisabled();
		expect(screen.getByLabelText(t('common.states.loading'))).toBeInTheDocument();
		await user().keyboard('{Escape}');
		expect(screen.getByRole('dialog')).toBeInTheDocument();
		resolve(makeTag({ name: 'ops' }));
		await vi.waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
		expect(toast.success).toHaveBeenCalledWith(t('tags.dialog.saved'));
	});

	test('editing without a change closes without a request', async () => {
		const onSaved = vi.fn();
		const { store, save } = open({ tag: makeTag({ id: 2, name: 'infra' }), onSaved });
		const update = vi.spyOn(store, 'update');
		await user().click(save);
		await vi.waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
		expect(update).not.toHaveBeenCalled();
		expect(onSaved).not.toHaveBeenCalled();
	});

	test('editing the colour only sends the colour and reports the saved tag', async () => {
		const onSaved = vi.fn();
		const tag = makeTag({ id: 2, name: 'infra', color: 'blue' });
		const { store, dialog, save } = open({ tag, onSaved });
		const saved = { ...tag, color: 'violet' as const };
		const update = vi.spyOn(store, 'update').mockResolvedValue(saved);
		await user().click(within(dialog).getByRole('radio', { name: t('tagColors.violet') }));
		await user().click(save);
		await vi.waitFor(() => expect(onSaved).toHaveBeenCalledWith(saved));
		expect(update).toHaveBeenCalledWith(2, { color: 'violet' });
	});

	test('cancel closes the dialog', async () => {
		const { dialog } = open();
		await user().click(within(dialog).getByRole('button', { name: t('common.actions.cancel') }));
		await vi.waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
	});
});
