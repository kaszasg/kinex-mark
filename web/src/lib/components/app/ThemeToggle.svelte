<script lang="ts">
	import Moon from '@lucide/svelte/icons/moon';
	import Sun from '@lucide/svelte/icons/sun';
	import { mode, setMode } from 'mode-watcher';
	import Button from './Button.svelte';
	import { t } from '$lib/i18n/index.js';
	import { preferences } from '$lib/state/preferences.svelte.js';
	import { session } from '$lib/state/session.svelte.js';
	import { cn } from '$lib/utils.js';
	import { buttonSize } from './styles.js';

	/**
	 * Sun/moon icon button (plan 8.7.1). Signed out it only sets the mode-watcher choice; signed
	 * in it also saves Preferences.theme.
	 */
	let { class: className }: { class?: string } = $props();

	const dark = $derived(mode.current !== 'light');

	function toggle(): void {
		const next = dark ? 'light' : 'dark';
		if (session.me) preferences.update({ theme: next });
		else setMode(next);
	}
</script>

<Button
	variant="ghost"
	size="icon"
	class={cn(buttonSize.icon, className)}
	aria-label={t('nav.theme.toggle')}
	onclick={toggle}
>
	{#if dark}
		<Sun class="size-4" strokeWidth={1.5} aria-hidden="true" />
	{:else}
		<Moon class="size-4" strokeWidth={1.5} aria-hidden="true" />
	{/if}
</Button>
