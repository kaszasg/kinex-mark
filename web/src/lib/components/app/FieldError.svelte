<script lang="ts">
	import CircleAlert from '@lucide/svelte/icons/circle-alert';
	import { fieldMessage, type Params } from '$lib/i18n/index.js';
	import { cn } from '$lib/utils.js';

	/**
	 * Error under a field: 14 px circle-alert and the text of fields.byField.<field>.<code>,
	 * otherwise fields.<code>. A ready `message` (for example a client-side check) wins.
	 * The id is referenced by the field's aria-describedby.
	 */
	type Props = {
		id: string;
		field?: string;
		code?: string | null;
		params?: Params;
		message?: string | null;
		class?: string;
	};
	let { id, field = '', code, params, message, class: className }: Props = $props();

	const text = $derived(message || (code ? fieldMessage(field, code, params) : ''));
</script>

{#if text}
	<p {id} class={cn('flex items-start gap-1.5 text-[13px] leading-5 text-destructive', className)}>
		<CircleAlert class="mt-[3px] size-3.5 shrink-0" strokeWidth={1.5} aria-hidden="true" />
		<span>{text}</span>
	</p>
{/if}
