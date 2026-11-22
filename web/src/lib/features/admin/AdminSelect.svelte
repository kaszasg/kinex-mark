<script lang="ts" module>
	export type SelectOption = { value: string; label: string; lang?: string; mono?: boolean };
	export type SelectGroup = { heading?: string; options: SelectOption[] };
</script>

<script lang="ts">
	import * as Select from '$lib/components/ui/select/index.js';
	import { inputSize } from '$lib/components/app/styles.js';
	import { cn } from '$lib/utils.js';

	/**
	 * Single-choice select of GRAPHICS.md 10.9 for the admin forms and filters: 36 px trigger with
	 * the field styling, options optionally in headed groups. The trigger takes the field id and
	 * the aria attributes of FormField, or an aria-label when it has no visible label.
	 */
	type Props = {
		id?: string;
		value: string;
		groups: SelectGroup[];
		disabled?: boolean;
		onchange: (value: string) => void;
		class?: string;
		'aria-label'?: string;
		'aria-invalid'?: 'true' | undefined;
		'aria-describedby'?: string | undefined;
	};
	let {
		id,
		value,
		groups,
		disabled = false,
		onchange,
		class: className,
		...aria
	}: Props = $props();

	const options = $derived(groups.flatMap((group) => group.options));
	const selected = $derived(options.find((option) => option.value === value));
</script>

<Select.Root
	type="single"
	{value}
	{disabled}
	onValueChange={(next: string) => {
		if (next !== value) onchange(next);
	}}
>
	<Select.Trigger
		{id}
		{...aria}
		class={cn(
			inputSize,
			// The trigger's own data-[size=default]:h-8 outranks h-9: without this the select is
			// 4 px lower than the inputs beside it (bottom-aligned filter rows shift its label).
			'w-full min-w-0 rounded-md bg-transparent data-[size=default]:h-9',
			className
		)}
	>
		<span class={cn('truncate', selected?.mono && 'font-mono')}>{selected?.label ?? ''}</span>
	</Select.Trigger>
	<Select.Content class="max-h-80">
		{#each groups as group, index (index)}
			<Select.Group>
				{#if group.heading}
					<Select.GroupHeading class="px-2 pt-2 pb-1 text-[11px] leading-4 text-muted-foreground">
						{group.heading}
					</Select.GroupHeading>
				{/if}
				{#each group.options as option (option.value)}
					<Select.Item
						value={option.value}
						label={option.label}
						lang={option.lang}
						class={cn('text-[13px]', option.mono && 'font-mono')}
					/>
				{/each}
			</Select.Group>
		{/each}
	</Select.Content>
</Select.Root>
