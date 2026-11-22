<script lang="ts" module>
	/** Preset periods of plan 8.7.6, in days. */
	export const ARCHIVE_PRESETS = [30, 60, 90, 180, 365] as const;
	/** Range of a custom period (plan 5.8: 0-3650, where 0 is "never" and has its own option). */
	export const CUSTOM_MIN = 1;
	export const CUSTOM_MAX = 3650;

	export type ArchiveOption = 'system' | 'never' | 'custom' | `${(typeof ARCHIVE_PRESETS)[number]}`;

	/** The option that shows a stored value: null is the system default, 0 is never. */
	export function optionFor(value: number | null): ArchiveOption {
		if (value === null) return 'system';
		if (value === 0) return 'never';
		const preset = ARCHIVE_PRESETS.find((days) => days === value);
		return preset === undefined ? 'custom' : (`${preset}` as ArchiveOption);
	}

	/** Parses a custom period; null when it is not a whole number in 1-3650. */
	export function parseCustomDays(text: string): number | null {
		const trimmed = text.trim();
		if (!/^\d{1,4}$/.test(trimmed)) return null;
		const days = Number(trimmed);
		return days >= CUSTOM_MIN && days <= CUSTOM_MAX ? days : null;
	}
</script>

<script lang="ts">
	import FormField from '$lib/components/app/FormField.svelte';
	import { inputSize } from '$lib/components/app/styles.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import * as Select from '$lib/components/ui/select/index.js';
	import { fmt, t } from '$lib/i18n/index.js';
	import { cn } from '$lib/utils.js';

	/**
	 * Automatic archiving period of plan 8.7.6: system default (with its current number of days),
	 * 30-365 days, never, or a custom number of days saved on Enter or when the field is left.
	 */
	type Props = {
		/** The stored preference: null follows the system default, 0 never archives. */
		value: number | null;
		/** The current system default (Preferences.systemArchiveAfterDays). */
		systemDays: number;
		onChange: (value: number | null) => void;
		/** Id of the select trigger, for the row label. */
		id: string;
		describedBy?: string;
		disabled?: boolean;
	};
	let { value, systemDays, onChange, id, describedBy, disabled = false }: Props = $props();

	/** True while the custom field is open for a value that is not custom yet. */
	let customRequested = $state(false);
	let draft = $state('');
	let draftCode = $state<string | null>(null);
	let customInput = $state<HTMLInputElement | null>(null);

	const stored = $derived(optionFor(value));
	const selected = $derived<ArchiveOption>(customRequested ? 'custom' : stored);
	const effective = $derived(value === null ? systemDays : value);

	const options = $derived<{ value: ArchiveOption; label: string }[]>([
		{ value: 'system', label: t('settings.display.archive.system', { count: systemDays }) },
		...ARCHIVE_PRESETS.map((days) => ({
			value: `${days}` as ArchiveOption,
			label: fmt.duration(days)
		})),
		{ value: 'never', label: t('settings.display.archive.never') },
		{ value: 'custom', label: t('settings.display.archive.custom') }
	]);
	const selectedLabel = $derived(options.find((option) => option.value === selected)?.label ?? '');

	// A new stored value (saved here, reverted, or changed elsewhere) closes a pending custom
	// request and shows a custom value in the field.
	let lastValue: number | null | undefined;
	$effect.pre(() => {
		const current = value;
		if (current === lastValue) return;
		lastValue = current;
		customRequested = false;
		if (current !== null && optionFor(current) === 'custom') {
			draft = String(current);
			draftCode = null;
		}
	});

	function choose(next: string): void {
		const option = options.find((item) => item.value === next)?.value;
		if (!option || option === selected) return;
		draftCode = null;
		if (option === 'custom') {
			customRequested = true;
			draft = effective > 0 ? String(effective) : '';
			queueMicrotask(() => customInput?.focus());
			return;
		}
		customRequested = false;
		if (option === 'system') onChange(null);
		else if (option === 'never') onChange(0);
		else onChange(Number(option));
	}

	function commit(): void {
		if (selected !== 'custom') return;
		if (draft.trim() === '') {
			draftCode = 'required';
			return;
		}
		const days = parseCustomDays(draft);
		if (days === null) {
			draftCode = 'out_of_range';
			return;
		}
		draftCode = null;
		if (days !== value) onChange(days);
	}

	function keydown(event: KeyboardEvent): void {
		if (event.key !== 'Enter') return;
		event.preventDefault();
		commit();
	}
</script>

<div class="flex w-full flex-col gap-3 sm:w-64">
	<Select.Root type="single" {disabled} bind:value={() => selected, choose}>
		<Select.Trigger {id} aria-describedby={describedBy} class={cn(inputSize, 'w-full')}>
			<span class="truncate">{selectedLabel}</span>
		</Select.Trigger>
		<Select.Content>
			{#each options as option (option.value)}
				<Select.Item value={option.value} label={option.label} />
			{/each}
		</Select.Content>
	</Select.Root>

	{#if selected === 'custom'}
		<FormField
			id={`${id}-custom`}
			label={t('settings.display.archive.customLabel')}
			field="archiveAfterDays"
			help={t('settings.display.archive.customHelp', { min: CUSTOM_MIN, max: CUSTOM_MAX })}
			errorCode={draftCode}
			errorParams={draftCode === 'out_of_range' ? { min: CUSTOM_MIN, max: CUSTOM_MAX } : undefined}
		>
			{#snippet children(control)}
				<Input
					{...control}
					bind:ref={customInput}
					bind:value={draft}
					inputmode="numeric"
					autocomplete="off"
					{disabled}
					class={cn(inputSize, 'w-28 font-mono')}
					onkeydown={keydown}
					onblur={commit}
				/>
			{/snippet}
		</FormField>
	{/if}

	<p class="text-[13px] leading-5 text-muted-foreground" aria-live="polite">
		{t('settings.display.archive.current', { count: effective })}
	</p>
</div>
