<script lang="ts" module>
	/** Attributes the control spreads to join the label, the help text and the error. */
	export type FieldControlProps = {
		id: string;
		'aria-invalid': 'true' | undefined;
		'aria-describedby': string | undefined;
	};
</script>

<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Label } from '$lib/components/ui/label/index.js';
	import type { Params } from '$lib/i18n/index.js';
	import { cn } from '$lib/utils.js';
	import FieldError from './FieldError.svelte';

	/**
	 * Label + control + help + FieldError (GRAPHICS.md 10.9). The control snippet receives the
	 * id, aria-invalid and aria-describedby attributes to spread onto the input.
	 */
	type Props = {
		id: string;
		label: string;
		/** Request field name for the per-field error texts (fields.byField.<field>). */
		field?: string;
		help?: string;
		errorCode?: string | null;
		errorParams?: Params;
		errorMessage?: string | null;
		class?: string;
		children: Snippet<[FieldControlProps]>;
	};
	let {
		id,
		label,
		field = '',
		help,
		errorCode,
		errorParams,
		errorMessage,
		class: className,
		children
	}: Props = $props();

	const helpId = $derived(`${id}-help`);
	const errorId = $derived(`${id}-error`);
	const invalid = $derived(Boolean(errorCode || errorMessage));
	const describedBy = $derived(
		[help ? helpId : null, invalid ? errorId : null].filter(Boolean).join(' ') || undefined
	);
	const controlProps = $derived<FieldControlProps>({
		id,
		'aria-invalid': invalid ? 'true' : undefined,
		'aria-describedby': describedBy
	});
</script>

<div class={cn('flex min-w-0 flex-col gap-1.5', className)}>
	<Label for={id} class="text-[13px] leading-5 font-medium">{label}</Label>
	{@render children(controlProps)}
	{#if help}
		<p id={helpId} class="text-[13px] leading-5 text-muted-foreground">{help}</p>
	{/if}
	<FieldError id={errorId} {field} code={errorCode} params={errorParams} message={errorMessage} />
</div>
