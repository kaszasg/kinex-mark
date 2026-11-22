<script lang="ts">
	import Eye from '@lucide/svelte/icons/eye';
	import EyeOff from '@lucide/svelte/icons/eye-off';
	import type { HTMLInputAttributes } from 'svelte/elements';
	import { Input } from '$lib/components/ui/input/index.js';
	import { t } from '$lib/i18n/index.js';
	import { cn } from '$lib/utils.js';
	import { inputSize } from './styles.js';

	/** Password field with a show/hide toggle (ghost icon button with aria-pressed). */
	type Props = Omit<HTMLInputAttributes, 'type' | 'value' | 'files'> & {
		value?: string;
		ref?: HTMLInputElement | null;
	};
	let {
		value = $bindable(''),
		ref = $bindable(null),
		class: className,
		disabled,
		...rest
	}: Props = $props();

	let visible = $state(false);
</script>

<div class="relative">
	<Input
		bind:ref
		bind:value
		type={visible ? 'text' : 'password'}
		autocapitalize="off"
		spellcheck={false}
		{disabled}
		class={cn(inputSize, 'pr-10', className)}
		{...rest}
	/>
	<button
		type="button"
		class="absolute inset-y-0 right-0 inline-flex w-9 items-center justify-center rounded-md text-muted-foreground transition-colors hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50"
		aria-label={visible ? t('common.password.hide') : t('common.password.show')}
		aria-pressed={visible}
		{disabled}
		onclick={() => (visible = !visible)}
	>
		{#if visible}
			<EyeOff class="size-4" strokeWidth={1.5} aria-hidden="true" />
		{:else}
			<Eye class="size-4" strokeWidth={1.5} aria-hidden="true" />
		{/if}
	</button>
</div>
