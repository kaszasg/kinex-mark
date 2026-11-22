<script lang="ts">
	import Globe from '@lucide/svelte/icons/globe';
	import type { Locale } from '$lib/api/types.js';
	import Button from './Button.svelte';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import * as ToggleGroup from '$lib/components/ui/toggle-group/index.js';
	import { i18n, isLocale, LOCALES, t } from '$lib/i18n/index.js';
	import { preferences } from '$lib/state/preferences.svelte.js';
	import { session } from '$lib/state/session.svelte.js';
	import { cn } from '$lib/utils.js';
	import { buttonSize, toggleItem } from './styles.js';

	/**
	 * Language switch of plan 8.4. `dropdown`: globe icon button with a radio menu, each language
	 * in its own language. `inline`: HU/EN/DE toggle group for the sign-in and gate pages.
	 * Signed in the choice is saved to the preferences; signed out it is an explicit choice that
	 * wins over the stored preference at sign-in (plan 8.5.1).
	 */
	type Props = { variant?: 'dropdown' | 'inline'; class?: string };
	let { variant = 'dropdown', class: className }: Props = $props();

	/** The name of a language in that language (the same in every catalogue). */
	const nameOf = (locale: Locale): string => t(`nav.locale.${locale}`);

	function change(next: string): void {
		if (!isLocale(next) || next === i18n.locale) return;
		if (session.me) preferences.update({ locale: next });
		else i18n.setLocale(next, { explicit: true });
	}
</script>

{#if variant === 'inline'}
	<ToggleGroup.Root
		type="single"
		bind:value={() => i18n.locale, change}
		aria-label={t('nav.locale.label')}
		class={cn('gap-0.5', className)}
	>
		{#each LOCALES as locale (locale)}
			<ToggleGroup.Item
				value={locale}
				lang={locale}
				aria-label={nameOf(locale)}
				class={cn(toggleItem, 'h-7 px-2 font-mono text-[11px] tracking-[0.08em] uppercase')}
			>
				{locale}
			</ToggleGroup.Item>
		{/each}
	</ToggleGroup.Root>
{:else}
	<DropdownMenu.Root>
		<DropdownMenu.Trigger>
			{#snippet child({ props })}
				<Button
					{...props}
					variant="ghost"
					size="icon"
					class={cn(buttonSize.icon, className)}
					aria-label={t('nav.locale.label')}
				>
					<Globe class="size-4" strokeWidth={1.5} aria-hidden="true" />
				</Button>
			{/snippet}
		</DropdownMenu.Trigger>
		<DropdownMenu.Content align="end" class="min-w-40">
			<DropdownMenu.RadioGroup value={i18n.locale} onValueChange={change}>
				{#each LOCALES as locale (locale)}
					<!-- Choosing a language closes the menu (the shadcn wrapper keeps it open by default). -->
					<DropdownMenu.RadioItem value={locale} lang={locale} closeOnSelect>
						{nameOf(locale)}
					</DropdownMenu.RadioItem>
				{/each}
			</DropdownMenu.RadioGroup>
		</DropdownMenu.Content>
	</DropdownMenu.Root>
{/if}
