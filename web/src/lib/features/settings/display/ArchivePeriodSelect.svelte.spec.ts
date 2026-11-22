// Automatic archiving period of plan 8.7.6: presets, never, system default and custom days.
import { render, screen } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { describe, expect, test, vi } from 'vitest';
import { fmt, t } from '$lib/i18n/index.js';
import ArchivePeriodSelect, {
	CUSTOM_MAX,
	CUSTOM_MIN,
	optionFor,
	parseCustomDays
} from './ArchivePeriodSelect.svelte';
import { normalizeSpace } from '$lib/test/text.js';

const user = () => userEvent.setup({ pointerEventsCheck: 0 });

/** toHaveTextContent collapses the received whitespace (NBSP included) but not the expected. */
const squash = (text: string): string => text.replace(/\s+/g, ' ');

// Floating content is visibility:hidden in jsdom: options are found by role and text.
function option(text: string): HTMLElement {
	const found = screen
		.queryAllByRole('option', { hidden: true })
		.find((el) => el.textContent?.trim() === text);
	if (!found) throw new Error(`no option "${text}"`);
	return found;
}

function setup(value: number | null, systemDays = 90) {
	const onChange = vi.fn();
	const view = render(ArchivePeriodSelect, { id: 'archive', value, systemDays, onChange });
	const trigger = screen.getByRole('combobox');
	return { onChange, trigger, view };
}

async function choose(trigger: HTMLElement, text: string): Promise<void> {
	const u = user();
	await u.click(trigger);
	await vi.waitFor(() => option(text));
	await u.click(option(text));
}

describe('helpers', () => {
	test('optionFor maps stored values to options', () => {
		expect(optionFor(null)).toBe('system');
		expect(optionFor(0)).toBe('never');
		expect(optionFor(180)).toBe('180');
		expect(optionFor(45)).toBe('custom');
	});

	test('parseCustomDays accepts whole numbers from 1 to 3650 only', () => {
		expect(parseCustomDays(' 45 ')).toBe(45);
		expect(parseCustomDays(String(CUSTOM_MIN))).toBe(1);
		expect(parseCustomDays(String(CUSTOM_MAX))).toBe(3650);
		expect(parseCustomDays('0')).toBeNull();
		expect(parseCustomDays('3651')).toBeNull();
		expect(parseCustomDays('4.5')).toBeNull();
		expect(parseCustomDays('-3')).toBeNull();
		expect(parseCustomDays('')).toBeNull();
	});
});

describe('ArchivePeriodSelect', () => {
	test('the system default shows its days and the effective period', () => {
		const { trigger } = setup(null, 120);
		expect(trigger).toHaveTextContent(
			normalizeSpace(t('settings.display.archive.system', { count: 120 }))
		);
		expect(
			screen.getByText(normalizeSpace(t('settings.display.archive.current', { count: 120 })))
		).toBeVisible();
	});

	test('a system default of 0 reads as never', () => {
		const { trigger } = setup(null, 0);
		expect(trigger).toHaveTextContent(t('settings.display.archive.system', { count: 0 }));
		expect(screen.getByText(t('settings.display.archive.current', { count: 0 }))).toBeVisible();
	});

	test('offers every option and saves a preset, never and the system default', async () => {
		const { trigger, onChange, view } = setup(90);
		expect(trigger).toHaveTextContent(squash(fmt.duration(90)));
		await user().click(trigger);
		await vi.waitFor(() => option(t('settings.display.archive.never')));
		const texts = screen
			.queryAllByRole('option', { hidden: true })
			.map((el) => el.textContent?.trim());
		expect(texts).toEqual([
			t('settings.display.archive.system', { count: 90 }),
			...[30, 60, 90, 180, 365].map((days) => fmt.duration(days)),
			t('settings.display.archive.never'),
			t('settings.display.archive.custom')
		]);
		await user().click(option(fmt.duration(30)));
		expect(onChange).toHaveBeenLastCalledWith(30);

		await view.rerender({ value: 30 });
		await choose(trigger, t('settings.display.archive.never'));
		expect(onChange).toHaveBeenLastCalledWith(0);

		await view.rerender({ value: 0 });
		expect(screen.getByText(t('settings.display.archive.current', { count: 0 }))).toBeVisible();
		await choose(trigger, t('settings.display.archive.system', { count: 90 }));
		expect(onChange).toHaveBeenLastCalledWith(null);
	});

	test('choosing the current option again saves nothing', async () => {
		const { trigger, onChange } = setup(90);
		await choose(trigger, fmt.duration(90));
		expect(onChange).not.toHaveBeenCalled();
	});

	test('custom opens a focused field prefilled with the effective days and saves on Enter', async () => {
		const { trigger, onChange, view } = setup(null, 90);
		await choose(trigger, t('settings.display.archive.custom'));
		const field = await screen.findByLabelText(t('settings.display.archive.customLabel'));
		await vi.waitFor(() => expect(field).toHaveFocus());
		expect(field).toHaveValue('90');
		expect(onChange).not.toHaveBeenCalled();
		await user().clear(field);
		await user().type(field, '45{Enter}');
		expect(onChange).toHaveBeenLastCalledWith(45);
		await view.rerender({ value: 45 });
		expect(trigger).toHaveTextContent(t('settings.display.archive.custom'));
		expect(screen.getByLabelText(t('settings.display.archive.customLabel'))).toHaveValue('45');
		expect(
			screen.getByText(normalizeSpace(t('settings.display.archive.current', { count: 45 })))
		).toBeVisible();
	});

	test('custom from never starts empty and needs a value', async () => {
		const { trigger, onChange } = setup(0);
		await choose(trigger, t('settings.display.archive.custom'));
		const field = await screen.findByLabelText(t('settings.display.archive.customLabel'));
		expect(field).toHaveValue('');
		await user().type(field, '{Enter}');
		expect(screen.getByText(t('fields.required'))).toBeInTheDocument();
		expect(onChange).not.toHaveBeenCalled();
	});

	test('an out-of-range custom value shows the range and saves nothing; blur saves a valid one', async () => {
		const { trigger, onChange } = setup(90);
		await choose(trigger, t('settings.display.archive.custom'));
		const field = await screen.findByLabelText(t('settings.display.archive.customLabel'));
		await user().clear(field);
		await user().type(field, '5000{Enter}');
		expect(field).toHaveAttribute('aria-invalid', 'true');
		expect(screen.getByText(t('fields.range', { min: CUSTOM_MIN, max: CUSTOM_MAX }))).toBeVisible();
		expect(onChange).not.toHaveBeenCalled();
		await user().clear(field);
		await user().type(field, '200');
		await user().tab();
		expect(onChange).toHaveBeenLastCalledWith(200);
	});

	test('a stored custom value opens the field; the same value is not saved again', async () => {
		const { trigger, onChange, view } = setup(45);
		expect(trigger).toHaveTextContent(t('settings.display.archive.custom'));
		const field = screen.getByLabelText(t('settings.display.archive.customLabel'));
		expect(field).toHaveValue('45');
		await user().type(field, '{Enter}');
		expect(onChange).not.toHaveBeenCalled();
		await view.rerender({ value: 180 });
		expect(trigger).toHaveTextContent(squash(fmt.duration(180)));
		expect(screen.queryByLabelText(t('settings.display.archive.customLabel'))).toBeNull();
	});

	test('a pending custom request closes when the stored value changes elsewhere', async () => {
		const { trigger, view } = setup(90);
		await choose(trigger, t('settings.display.archive.custom'));
		expect(await screen.findByLabelText(t('settings.display.archive.customLabel'))).toBeVisible();
		await view.rerender({ value: 60 });
		expect(trigger).toHaveTextContent(squash(fmt.duration(60)));
		expect(screen.queryByLabelText(t('settings.display.archive.customLabel'))).toBeNull();
	});
});
