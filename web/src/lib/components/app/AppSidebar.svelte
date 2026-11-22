<script lang="ts">
	import Bookmark from '@lucide/svelte/icons/bookmark';
	import ChevronUp from '@lucide/svelte/icons/chevron-up';
	import FileText from '@lucide/svelte/icons/file-text';
	import LogOut from '@lucide/svelte/icons/log-out';
	import ScrollText from '@lucide/svelte/icons/scroll-text';
	import Settings from '@lucide/svelte/icons/settings';
	import SlidersHorizontal from '@lucide/svelte/icons/sliders-horizontal';
	import TagsIcon from '@lucide/svelte/icons/tags';
	import Users from '@lucide/svelte/icons/users';
	import type { Component } from 'svelte';
	import { page } from '$app/state';
	import Logo from '$lib/components/brand/Logo.svelte';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import * as Sidebar from '$lib/components/ui/sidebar/index.js';
	import { fmt, t, type MessageKey } from '$lib/i18n/index.js';
	import { linkStats } from '$lib/state/link-stats.svelte.js';
	import { session } from '$lib/state/session.svelte.js';
	import { tags } from '$lib/state/tags.svelte.js';
	import { tagSwatchClass } from '$lib/tags/colors.js';
	import { cn } from '$lib/utils.js';
	import { typeEyebrow } from './styles.js';

	/** Sidebar content of plan 8.7.1; `onNavigate` closes the off-canvas sheet after a click. */
	type Props = { onNavigate?: () => void; class?: string };
	let { onNavigate, class: className }: Props = $props();

	type Item = {
		href: string;
		label: MessageKey;
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		icon: Component<any>;
		count?: number | null;
		exact?: boolean;
	};

	const pathname = $derived(page.url.pathname);
	const me = $derived(session.me);

	const collection = $derived<Item[]>([
		{ href: '/', label: 'nav.links', icon: Bookmark, count: linkStats.active, exact: true },
		{
			href: '/tags',
			label: 'nav.tags',
			icon: TagsIcon,
			count: tags.status === 'ready' ? tags.items.length : null
		}
	]);
	const account: Item[] = [{ href: '/settings', label: 'nav.settings', icon: Settings }];
	const admin: Item[] = [
		{ href: '/admin/users', label: 'nav.adminUsers', icon: Users },
		{ href: '/admin/settings', label: 'nav.adminSettings', icon: SlidersHorizontal },
		{ href: '/admin/audit', label: 'nav.adminAudit', icon: ScrollText }
	];

	function isActive(item: Item): boolean {
		if (item.exact) return pathname === item.href;
		return pathname === item.href || pathname.startsWith(`${item.href}/`);
	}

	const buttonClass =
		'h-8 text-[13px] leading-5 text-sidebar-foreground data-active:font-medium [&[data-active]>svg]:text-sidebar-primary';
	const counterClass = 'font-mono text-[11px] text-muted-foreground';
</script>

{#snippet group(label: MessageKey, items: Item[])}
	<Sidebar.Group class="py-1">
		<Sidebar.GroupLabel class={cn(typeEyebrow, 'h-7 px-2')}>
			{t(label)}
		</Sidebar.GroupLabel>
		<Sidebar.GroupContent>
			<Sidebar.Menu>
				{#each items as item (item.href)}
					{@const active = isActive(item)}
					<Sidebar.MenuItem>
						<Sidebar.MenuButton isActive={active} class={buttonClass}>
							{#snippet child({ props })}
								<a
									{...props}
									href={item.href}
									aria-current={active ? 'page' : undefined}
									onclick={() => onNavigate?.()}
								>
									<item.icon strokeWidth={1.5} aria-hidden="true" />
									<span>{t(item.label)}</span>
								</a>
							{/snippet}
						</Sidebar.MenuButton>
						{#if item.count !== undefined && item.count !== null}
							<Sidebar.MenuBadge class={counterClass}>{fmt.number(item.count)}</Sidebar.MenuBadge>
						{/if}
					</Sidebar.MenuItem>
				{/each}
			</Sidebar.Menu>
		</Sidebar.GroupContent>
	</Sidebar.Group>
{/snippet}

<Sidebar.Root
	collapsible="none"
	class={cn('h-full w-full border-r border-sidebar-border', className)}
>
	<Sidebar.Header class="px-2 pt-3 pb-0">
		<a
			href="/"
			class="mb-3 flex h-10 items-center gap-2 rounded-md px-2 text-sidebar-foreground focus-visible:outline-2 focus-visible:outline-ring"
			onclick={() => onNavigate?.()}
		>
			<Logo variant="mark" height={28} class="text-sidebar-foreground" />
			<span class="text-[15px] leading-6 font-medium">{t('common.appName')}</span>
		</a>
	</Sidebar.Header>
	<!-- The groups are the main navigation of the application (WCAG 1.3.1 landmark). -->
	<Sidebar.Content class="gap-1 px-0">
		<nav aria-label={t('nav.sidebarTitle')} class="flex flex-col gap-1">
			{@render group('nav.groups.collection', collection)}
			{#if tags.frequent.length > 0}
				<Sidebar.Group class="py-1">
					<Sidebar.GroupLabel class={cn(typeEyebrow, 'h-7 px-2')}>
						{t('nav.groups.frequentTags')}
					</Sidebar.GroupLabel>
					<Sidebar.GroupContent>
						<Sidebar.Menu>
							{#each tags.frequent as tag (tag.id)}
								<Sidebar.MenuItem>
									<Sidebar.MenuButton class={buttonClass}>
										{#snippet child({ props })}
											<a {...props} href="/?tags={tag.id}" onclick={() => onNavigate?.()}>
												<span
													class="ml-1 size-2 shrink-0 {tagSwatchClass[tag.color]}"
													aria-hidden="true"
												></span>
												<span>{tag.name}</span>
											</a>
										{/snippet}
									</Sidebar.MenuButton>
									<Sidebar.MenuBadge class={counterClass}>
										{fmt.number(tag.activeLinkCount)}
									</Sidebar.MenuBadge>
								</Sidebar.MenuItem>
							{/each}
						</Sidebar.Menu>
					</Sidebar.GroupContent>
				</Sidebar.Group>
			{/if}
			{@render group('nav.groups.account', account)}
			{#if session.can('admin')}
				{@render group('nav.groups.admin', admin)}
			{/if}
		</nav>
	</Sidebar.Content>
	{#if me}
		<Sidebar.Footer class="border-t border-sidebar-border p-2">
			<Sidebar.Menu>
				<Sidebar.MenuItem>
					<DropdownMenu.Root>
						<DropdownMenu.Trigger>
							{#snippet child({ props })}
								<Sidebar.MenuButton
									{...props}
									size="lg"
									class="h-12 text-sidebar-foreground"
									aria-label={t('nav.userMenu', {
										name: me.displayName || me.username,
										role: t(`roles.${me.role}`)
									})}
								>
									<span class="flex min-w-0 flex-1 flex-col text-left">
										<span class="truncate text-[13px] leading-5 font-medium">
											{me.displayName || me.username}
										</span>
										<span class="truncate text-[11px] leading-4 text-muted-foreground">
											{t(`roles.${me.role}`)}
										</span>
									</span>
									<ChevronUp class="ml-auto size-4" strokeWidth={1.5} aria-hidden="true" />
								</Sidebar.MenuButton>
							{/snippet}
						</DropdownMenu.Trigger>
						<DropdownMenu.Content
							side="top"
							align="start"
							class="w-(--bits-dropdown-menu-anchor-width) min-w-52"
						>
							<DropdownMenu.Item onSelect={() => onNavigate?.()}>
								{#snippet child({ props })}
									<a href="/settings" {...props}>
										<Settings class="size-4" strokeWidth={1.5} aria-hidden="true" />
										{t('nav.settings')}
									</a>
								{/snippet}
							</DropdownMenu.Item>
							<DropdownMenu.Item onSelect={() => onNavigate?.()}>
								{#snippet child({ props })}
									<a href="/privacy" {...props}>
										<FileText class="size-4" strokeWidth={1.5} aria-hidden="true" />
										{t('nav.privacyNotice')}
									</a>
								{/snippet}
							</DropdownMenu.Item>
							<DropdownMenu.Separator />
							<DropdownMenu.Item onSelect={() => void session.logout()}>
								<LogOut class="size-4" strokeWidth={1.5} aria-hidden="true" />
								{t('nav.logout')}
							</DropdownMenu.Item>
						</DropdownMenu.Content>
					</DropdownMenu.Root>
				</Sidebar.MenuItem>
			</Sidebar.Menu>
		</Sidebar.Footer>
	{/if}
</Sidebar.Root>
