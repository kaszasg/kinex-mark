<script lang="ts" module>
	/** Directory search waits this long after the last keystroke. */
	export const DIRECTORY_DEBOUNCE_MS = 200;
	/** The directory accepts at most this many runes in q (plan 5.11). */
	export const DIRECTORY_QUERY_MAX = 64;
</script>

<script lang="ts">
	import ChevronsUpDown from '@lucide/svelte/icons/chevrons-up-down';
	import Globe from '@lucide/svelte/icons/globe';
	import UserIcon from '@lucide/svelte/icons/user';
	import Users from '@lucide/svelte/icons/users';
	import X from '@lucide/svelte/icons/x';
	import { onDestroy, untrack } from 'svelte';
	import { toast } from 'svelte-sonner';
	import type { ApiError } from '$lib/api/errors.js';
	import * as directoryApi from '$lib/api/directory.js';
	import * as linksApi from '$lib/api/links.js';
	import type {
		DirectoryUser,
		LinkItem,
		Role,
		Share,
		ShareTarget,
		ShareTargetType
	} from '$lib/api/types.js';
	import ErrorAlert from '$lib/components/app/ErrorAlert.svelte';
	import FieldError from '$lib/components/app/FieldError.svelte';
	import LoadingRows from '$lib/components/app/LoadingRows.svelte';
	import {
		buttonSize,
		dialogBox,
		dialogFooter,
		inputSize,
		toggleItem,
		typeBodySm,
		typeHeading3
	} from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import * as Command from '$lib/components/ui/command/index.js';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import * as Popover from '$lib/components/ui/popover/index.js';
	import * as Select from '$lib/components/ui/select/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import * as ToggleGroup from '$lib/components/ui/toggle-group/index.js';
	import { errorMessage, t } from '$lib/i18n/index.js';
	import { links as linksStore, toApiError, type LinksStore } from '$lib/state/links.svelte.js';
	import { debounce } from '$lib/utils/debounce.js';
	import { cleanQueryText } from '$lib/utils/text.js';
	import { cn } from '$lib/utils.js';
	import { toastError } from '$lib/utils/toast-error.js';

	/**
	 * Sharing of an own link (plan 8.7.3, 5.9.6): the current shares with revoke buttons and a
	 * new share for everyone, a role or one user from the directory.
	 */
	type Props = { open?: boolean; link: LinkItem | null; store?: LinksStore };
	let { open = $bindable(false), link, store = linksStore }: Props = $props();

	const ROLES: Role[] = ['admin', 'developer', 'customer'];
	const TYPES: ShareTargetType[] = ['all', 'role', 'user'];

	let shares = $state<Share[]>([]);
	let loadStatus = $state<'loading' | 'ready' | 'error'>('loading');
	let loadError = $state<ApiError | null>(null);
	let targetType = $state<ShareTargetType>('all');
	let role = $state<Role | ''>('');
	let user = $state<DirectoryUser | null>(null);
	let userOpen = $state(false);
	let userQuery = $state('');
	let directory = $state<DirectoryUser[]>([]);
	let directoryLoading = $state(false);
	let adding = $state(false);
	let revoking = $state<number | null>(null);
	let addError = $state<ApiError | null>(null);
	let pickerCode = $state<string | null>(null);
	let pickerMessage = $state<string | null>(null);
	let directoryAbort: AbortController | null = null;

	async function load(id: number): Promise<void> {
		loadStatus = 'loading';
		loadError = null;
		try {
			shares = (await linksApi.listShares(id)).items;
			loadStatus = 'ready';
		} catch (err) {
			loadError = toApiError(err);
			loadStatus = 'error';
		}
	}

	$effect(() => {
		if (!open || !link) return;
		const id = link.id;
		untrack(() => {
			targetType = 'all';
			role = '';
			user = null;
			userQuery = '';
			addError = null;
			pickerCode = null;
			pickerMessage = null;
			void load(id);
		});
	});

	async function searchUsers(q: string): Promise<void> {
		directoryAbort?.abort();
		const controller = new AbortController();
		directoryAbort = controller;
		directoryLoading = true;
		try {
			const result = await directoryApi.searchDirectory(
				cleanQueryText(q, DIRECTORY_QUERY_MAX),
				controller.signal
			);
			if (controller === directoryAbort) directory = result.items;
		} catch (err) {
			if ((err as { name?: string })?.name !== 'AbortError' && controller === directoryAbort) {
				directory = [];
				toastError(err);
			}
		} finally {
			if (controller === directoryAbort) directoryLoading = false;
		}
	}

	const debouncedSearch = debounce((q: string) => void searchUsers(q), DIRECTORY_DEBOUNCE_MS);
	onDestroy(() => {
		debouncedSearch.cancel();
		directoryAbort?.abort();
	});

	// Every opening of the user picker starts with an empty search and the full first page.
	$effect(() => {
		if (!userOpen) return;
		untrack(() => {
			userQuery = '';
			debouncedSearch.cancel();
			void searchUsers('');
		});
	});

	function shareText(share: Share): string {
		if (share.targetType === 'all') return t('links.share.everyone');
		if (share.targetType === 'role' && share.targetRole) {
			return t('links.share.roleTarget', { role: t(`roles.plural.${share.targetRole}`) });
		}
		return share.targetUser?.displayName || share.targetUser?.username || t('common.missing');
	}

	function syncShareCount(count: number): void {
		if (!link) return;
		const current = store.find(link.id);
		if (current) store.upsert({ ...current, shareCount: count });
	}

	function setType(value: string): void {
		if ((TYPES as string[]).includes(value)) {
			targetType = value as ShareTargetType;
			pickerCode = null;
			pickerMessage = null;
		}
	}

	async function add(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (!link || adding) return;
		addError = null;
		pickerCode = null;
		pickerMessage = null;
		const target: ShareTarget = { targetType };
		if (targetType === 'role') {
			if (!role) {
				pickerCode = 'required';
				return;
			}
			target.targetRole = role;
		} else if (targetType === 'user') {
			if (!user) {
				pickerCode = 'required';
				return;
			}
			target.targetUserId = user.id;
		}
		adding = true;
		try {
			const share = await linksApi.createShare(link.id, target);
			shares = [...shares, share];
			user = null;
			role = '';
			syncShareCount(shares.length);
			toast.success(t('links.toast.shared'));
		} catch (err) {
			const apiError = toApiError(err);
			const fieldCode = apiError.field('targetRole') ?? apiError.field('targetUserId');
			if (apiError.is('share_duplicate')) pickerMessage = errorMessage(apiError);
			else if (apiError.is('validation_failed') && fieldCode) pickerCode = fieldCode;
			else addError = apiError;
		} finally {
			adding = false;
		}
	}

	async function revoke(share: Share): Promise<void> {
		if (!link || revoking !== null) return;
		revoking = share.id;
		try {
			await linksApi.deleteShare(link.id, share.id);
			shares = shares.filter((item) => item.id !== share.id);
			syncShareCount(shares.length);
			toast.success(t('links.toast.unshared'));
		} catch (err) {
			toastError(err);
		} finally {
			revoking = null;
		}
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Content class={dialogBox} showCloseButton={false}>
		<Dialog.Header class="gap-1.5 text-left">
			<Dialog.Title class={typeHeading3}>{t('links.share.title')}</Dialog.Title>
			<Dialog.Description class={cn(typeBodySm, 'text-muted-foreground')}>
				{t('links.share.text')}
			</Dialog.Description>
		</Dialog.Header>

		<section class="flex flex-col gap-2" aria-labelledby="share-current-heading">
			<h3 id="share-current-heading" class="text-[13px] leading-5 font-semibold">
				{t('links.share.current')}
			</h3>
			{#if loadStatus === 'loading'}
				<LoadingRows rows={2} rowClass="h-8" />
			{:else if loadStatus === 'error'}
				<ErrorAlert error={loadError} onRetry={() => link && void load(link.id)} />
			{:else if shares.length === 0}
				<p class="body-sm text-muted-foreground">{t('links.share.empty')}</p>
			{:else}
				<ul class="flex flex-col rounded-md border">
					{#each shares as share (share.id)}
						{@const text = shareText(share)}
						<li class="flex min-h-10 items-center gap-2 border-b px-3 py-1.5 last:border-b-0">
							{#if share.targetType === 'all'}
								<Globe class="size-4 text-muted-foreground" strokeWidth={1.5} aria-hidden="true" />
							{:else if share.targetType === 'role'}
								<Users class="size-4 text-muted-foreground" strokeWidth={1.5} aria-hidden="true" />
							{:else}
								<UserIcon
									class="size-4 text-muted-foreground"
									strokeWidth={1.5}
									aria-hidden="true"
								/>
							{/if}
							<span class="min-w-0 flex-1 truncate text-[13px] leading-5">
								{text}
								{#if share.targetUser}
									<span class="ml-1 font-mono text-muted-foreground">
										{share.targetUser.username}
									</span>
								{/if}
							</span>
							<Button
								variant="ghost"
								size="icon"
								class={buttonSize.iconSm}
								aria-label={t('links.share.revoke', { target: text })}
								disabled={revoking !== null}
								onclick={() => void revoke(share)}
							>
								{#if revoking === share.id}
									<Spinner class="size-4" aria-label={t('common.states.loading')} />
								{:else}
									<X class="size-4" strokeWidth={1.5} aria-hidden="true" />
								{/if}
							</Button>
						</li>
					{/each}
				</ul>
			{/if}
		</section>

		<form class="flex flex-col gap-3" onsubmit={add} aria-labelledby="share-new-heading">
			<h3 id="share-new-heading" class="text-[13px] leading-5 font-semibold">
				{t('links.share.newShare')}
			</h3>
			<ToggleGroup.Root
				type="single"
				variant="outline"
				bind:value={() => targetType, setType}
				aria-label={t('links.share.type.label')}
				class="self-start"
			>
				{#each TYPES as type (type)}
					<ToggleGroup.Item value={type} class={cn(toggleItem, 'h-9 px-3 text-[13px]')}>
						{t(`links.share.type.${type}`)}
					</ToggleGroup.Item>
				{/each}
			</ToggleGroup.Root>

			{#if targetType === 'role'}
				<div class="flex flex-col gap-1.5">
					<Label for="share-role" class="text-[13px] leading-5 font-medium">
						{t('links.share.role')}
					</Label>
					<Select.Root
						type="single"
						value={role}
						onValueChange={(value: string) => {
							role = (ROLES as string[]).includes(value) ? (value as Role) : '';
							pickerCode = null;
							pickerMessage = null;
						}}
					>
						<Select.Trigger
							id="share-role"
							class={cn(inputSize, 'w-full')}
							aria-invalid={pickerCode || pickerMessage ? 'true' : undefined}
							aria-describedby="share-picker-error"
						>
							{role ? t(`roles.plural.${role}`) : t('links.share.rolePlaceholder')}
						</Select.Trigger>
						<Select.Content>
							{#each ROLES as item (item)}
								<Select.Item value={item} label={t(`roles.plural.${item}`)} />
							{/each}
						</Select.Content>
					</Select.Root>
				</div>
			{:else if targetType === 'user'}
				<div class="flex flex-col gap-1.5">
					<Label for="share-user" class="text-[13px] leading-5 font-medium">
						{t('links.share.user')}
					</Label>
					<Popover.Root bind:open={userOpen}>
						<Popover.Trigger id="share-user">
							{#snippet child({ props })}
								<Button
									{...props}
									variant="outline"
									class={cn(buttonSize.default, 'w-full justify-between font-normal')}
									aria-invalid={pickerCode || pickerMessage ? 'true' : undefined}
									aria-describedby="share-picker-error"
								>
									{#if user}
										<span class="truncate">
											{user.displayName || user.username}
											<span class="ml-1 font-mono text-muted-foreground">{user.username}</span>
										</span>
									{:else}
										<span class="text-muted-foreground">{t('links.share.userPlaceholder')}</span>
									{/if}
									<ChevronsUpDown
										class="size-4 text-muted-foreground"
										strokeWidth={1.5}
										aria-hidden="true"
									/>
								</Button>
							{/snippet}
						</Popover.Trigger>
						<Popover.Content class="w-[min(392px,calc(100vw-48px))] p-0" align="start">
							<Command.Root shouldFilter={false}>
								<Command.Input
									bind:value={userQuery}
									placeholder={t('links.share.userSearch')}
									aria-label={t('links.share.userSearch')}
									maxlength={DIRECTORY_QUERY_MAX}
									oninput={() => debouncedSearch(userQuery)}
								/>
								<Command.List aria-label={t('links.share.user')}>
									{#if directoryLoading && directory.length === 0}
										<div class="flex justify-center py-4">
											<Spinner class="size-4" aria-label={t('common.states.loading')} />
										</div>
									{:else if directory.length === 0}
										<p class="px-3 py-6 text-center text-[13px] leading-5 text-muted-foreground">
											{t('links.share.userEmpty')}
										</p>
									{:else}
										<Command.Group>
											{#each directory as entry (entry.id)}
												<Command.Item
													value={String(entry.id)}
													onSelect={() => {
														user = entry;
														userOpen = false;
														pickerCode = null;
														pickerMessage = null;
													}}
												>
													<span class="min-w-0 flex-1 truncate">
														{entry.displayName || entry.username}
													</span>
													<span class="font-mono text-[11px] text-muted-foreground">
														{entry.username}
													</span>
												</Command.Item>
											{/each}
										</Command.Group>
									{/if}
								</Command.List>
							</Command.Root>
						</Popover.Content>
					</Popover.Root>
				</div>
			{/if}
			<FieldError
				id="share-picker-error"
				field={targetType === 'role' ? 'targetRole' : 'targetUserId'}
				code={pickerCode}
				message={pickerMessage}
			/>
			{#if addError}
				<ErrorAlert error={addError} />
			{/if}
			<Dialog.Footer class={dialogFooter}>
				<Button
					type="button"
					variant="outline"
					class={buttonSize.default}
					onclick={() => (open = false)}
				>
					{t('common.actions.done')}
				</Button>
				<Button
					type="submit"
					class={buttonSize.default}
					disabled={adding || loadStatus !== 'ready'}
					aria-busy={adding}
				>
					{#if adding}<Spinner class="size-4" aria-label={t('common.states.loading')} />{/if}
					{t('links.share.add')}
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
