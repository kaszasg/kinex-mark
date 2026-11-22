<script lang="ts">
	import Search from '@lucide/svelte/icons/search';
	import UserPlus from '@lucide/svelte/icons/user-plus';
	import Users from '@lucide/svelte/icons/users';
	import { untrack } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { adminApi } from '$lib/api/index.js';
	import type { AdminUser, AdminUserCreated, Page } from '$lib/api/types.js';
	import EmptyState from '$lib/components/app/EmptyState.svelte';
	import ErrorAlert from '$lib/components/app/ErrorAlert.svelte';
	import LoadingRows from '$lib/components/app/LoadingRows.svelte';
	import PageHeader from '$lib/components/app/PageHeader.svelte';
	import StateAlert from '$lib/components/app/StateAlert.svelte';
	import { buttonSize, inputSize } from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import { Input } from '$lib/components/ui/input/index.js';
	import { t } from '$lib/i18n/index.js';
	import { session } from '$lib/state/session.svelte.js';
	import { debounce } from '$lib/utils/debounce.js';
	import AdminSelect from './AdminSelect.svelte';
	import Pager from './Pager.svelte';
	import StepUpDialog from './StepUpDialog.svelte';
	import TempPasswordDialog from './TempPasswordDialog.svelte';
	import UserDialog from './UserDialog.svelte';
	import UsersTable, { type UserAction } from './UsersTable.svelte';
	import {
		DEFAULT_USERS_FILTER,
		hasUserFilters,
		parseUsersFilter,
		ROLES,
		usersApiQuery,
		usersFilterSearch,
		type UsersFilter
	} from './users-query.js';
	import { toastError } from '$lib/utils/toast-error.js';

	/**
	 * The /admin/users screen of plan 8.7.7: heading with the add button, filters kept in the URL,
	 * the users table with its row actions, paging, and the create, edit, step-up and
	 * temporary-password dialogs.
	 */

	const SEARCH_DEBOUNCE_MS = 200;
	const ALL = 'all';

	type ConfirmKind = 'resetPassword' | 'resetTwoFactor' | 'deactivate' | 'activate' | 'delete';

	const filter = $derived(parseUsersFilter(page.url.searchParams));
	const selfId = $derived(session.me?.id ?? null);

	let data = $state<Page<AdminUser> | null>(null);
	let error = $state<unknown>(null);
	let loading = $state(false);
	let search = $state('');
	let busyId = $state<number | null>(null);
	let requestSeq = 0;

	let dialogOpen = $state(false);
	let dialogUser = $state<AdminUser | null>(null);
	let confirmOpen = $state(false);
	let confirmKind = $state<ConfirmKind>('resetPassword');
	let confirmUser = $state<AdminUser | null>(null);
	let tempOpen = $state(false);
	let tempUsername = $state('');
	let tempPassword = $state('');

	async function load(next: UsersFilter): Promise<void> {
		const seq = ++requestSeq;
		loading = true;
		error = null;
		try {
			const result = await adminApi.listUsers(usersApiQuery(next));
			if (seq === requestSeq) data = result;
		} catch (err) {
			if (seq === requestSeq) error = err;
		} finally {
			if (seq === requestSeq) loading = false;
		}
	}

	const reload = (): Promise<void> => load(filter);

	$effect(() => {
		const next = filter;
		untrack(() => {
			search = next.q;
			void load(next);
		});
	});

	function navigate(next: UsersFilter, replace = false): void {
		void goto(`${page.url.pathname}${usersFilterSearch(next)}`, { replace, reset: false });
	}

	function setFilter(patch: Partial<UsersFilter>): void {
		navigate({ ...filter, ...patch, page: patch.page ?? 1 });
	}

	const searchLater = debounce((value: string) => {
		const q = value.trim();
		if (q !== filter.q) navigate({ ...filter, q, page: 1 }, true);
	}, SEARCH_DEBOUNCE_MS);

	$effect(() => () => searchLater.cancel());

	const roleGroups = $derived([
		{ options: [{ value: ALL, label: t('admin.users.filters.allRoles') }] },
		{ options: ROLES.map((value) => ({ value, label: t(`roles.${value}`) })) }
	]);
	const statusGroups = $derived([
		{
			options: [
				{ value: 'active', label: t('admin.users.filters.statusActive') },
				{ value: 'inactive', label: t('admin.users.filters.statusInactive') },
				{ value: 'all', label: t('admin.users.filters.statusAll') }
			]
		}
	]);

	function openCreate(): void {
		dialogUser = null;
		dialogOpen = true;
	}

	function showTempPassword(username: string, password: string | null): void {
		if (!password) return;
		tempUsername = username;
		tempPassword = password;
		tempOpen = true;
	}

	function created(result: AdminUserCreated): void {
		toast.success(t('admin.users.toast.created', { username: result.user.username }));
		showTempPassword(result.user.username, result.temporaryPassword);
		void reload();
	}

	function saved(user: AdminUser): void {
		toast.success(t('admin.users.toast.updated', { username: user.username }));
		void reload();
	}

	async function direct(user: AdminUser, run: () => Promise<string>): Promise<void> {
		busyId = user.id;
		try {
			toast.success(await run());
			void reload();
		} catch (err) {
			toastError(err);
		} finally {
			busyId = null;
		}
	}

	function onaction(action: UserAction, user: AdminUser): void {
		switch (action) {
			case 'edit':
				dialogUser = user;
				dialogOpen = true;
				return;
			case 'unlock':
				void direct(user, async () => {
					await adminApi.unlockUser(user.id);
					return t('admin.users.toast.unlocked', { username: user.username });
				});
				return;
			case 'revokeSessions':
				void direct(user, async () => {
					const { revoked } = await adminApi.revokeUserSessions(user.id);
					return t('admin.users.toast.sessionsRevoked', { count: revoked });
				});
				return;
			default:
				confirmKind = action;
				confirmUser = user;
				confirmOpen = true;
		}
	}

	const confirmTexts = $derived.by(() => {
		const username = confirmUser?.username ?? '';
		switch (confirmKind) {
			case 'resetPassword':
				return {
					title: t('admin.users.confirm.resetPassword.title', { username }),
					text: t('admin.users.confirm.resetPassword.text'),
					action: t('admin.users.confirm.resetPassword.action'),
					destructive: false
				};
			case 'resetTwoFactor':
				return {
					title: t('admin.users.confirm.resetTwoFactor.title', { username }),
					text: t('admin.users.confirm.resetTwoFactor.text'),
					action: t('admin.users.confirm.resetTwoFactor.action'),
					destructive: false
				};
			case 'deactivate':
				return {
					title: t('admin.users.confirm.deactivate.title', { username }),
					text: t('admin.users.confirm.deactivate.text'),
					action: t('admin.users.confirm.deactivate.action'),
					destructive: true
				};
			case 'activate':
				return {
					title: t('admin.users.confirm.activate.title', { username }),
					text: t('admin.users.confirm.activate.text'),
					action: t('admin.users.confirm.activate.action'),
					destructive: false
				};
			default:
				return {
					title: t('admin.users.confirm.delete.title', { username }),
					text: t('admin.users.confirm.delete.text'),
					action: t('admin.users.confirm.delete.action'),
					destructive: true
				};
		}
	});

	async function confirmAction(currentPassword: string): Promise<void> {
		const user = confirmUser;
		if (!user) return;
		const username = user.username;
		switch (confirmKind) {
			case 'resetPassword': {
				const { temporaryPassword } = await adminApi.resetPassword(user.id, currentPassword);
				toast.success(t('admin.users.toast.passwordReset', { username }));
				showTempPassword(username, temporaryPassword);
				break;
			}
			case 'resetTwoFactor':
				await adminApi.resetTwoFactor(user.id, currentPassword);
				toast.success(t('admin.users.toast.twoFactorReset', { username }));
				break;
			case 'deactivate':
				await adminApi.updateUser(user.id, { active: false, currentPassword });
				toast.success(t('admin.users.toast.deactivated', { username }));
				break;
			case 'activate':
				await adminApi.updateUser(user.id, { active: true, currentPassword });
				toast.success(t('admin.users.toast.activated', { username }));
				break;
			default:
				await adminApi.deleteUser(user.id, currentPassword);
				toast.success(t('admin.users.toast.deleted', { username }));
		}
		void reload();
	}
</script>

<PageHeader
	title={t('admin.users.pageTitle')}
	subtitle={data ? t('admin.users.summary', { count: data.total }) : undefined}
>
	{#snippet actions()}
		<Button class={buttonSize.default} onclick={openCreate}>
			<UserPlus class="size-4" strokeWidth={1.5} aria-hidden="true" />
			{t('admin.users.add')}
		</Button>
	{/snippet}
</PageHeader>

<div
	role="search"
	class="flex flex-wrap items-end gap-3"
	aria-label={t('admin.users.filters.label')}
>
	<div class="flex w-full flex-col gap-1.5 sm:w-64">
		<label for="users-search" class="text-[13px] leading-5 font-medium">
			{t('admin.users.filters.search')}
		</label>
		<div class="relative">
			<Search
				class="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground"
				strokeWidth={1.5}
				aria-hidden="true"
			/>
			<Input
				id="users-search"
				type="search"
				class="{inputSize} pl-9"
				placeholder={t('admin.users.filters.searchPlaceholder')}
				maxlength={64}
				value={search}
				oninput={(event: Event) => {
					search = (event.currentTarget as HTMLInputElement).value;
					searchLater(search);
				}}
				onkeydown={(event: KeyboardEvent) => {
					if (event.key === 'Enter') searchLater.flush();
				}}
			/>
		</div>
	</div>
	<div class="flex w-[calc(50%-6px)] flex-col gap-1.5 sm:w-44">
		<label for="users-role" class="text-[13px] leading-5 font-medium">
			{t('admin.users.filters.role')}
		</label>
		<AdminSelect
			id="users-role"
			value={filter.role || ALL}
			groups={roleGroups}
			onchange={(value) => setFilter({ role: value === ALL ? '' : (value as UsersFilter['role']) })}
		/>
	</div>
	<div class="flex w-[calc(50%-6px)] flex-col gap-1.5 sm:w-36">
		<label for="users-status" class="text-[13px] leading-5 font-medium">
			{t('admin.users.filters.status')}
		</label>
		<AdminSelect
			id="users-status"
			value={filter.status}
			groups={statusGroups}
			onchange={(value) => setFilter({ status: value as UsersFilter['status'] })}
		/>
	</div>
	{#if hasUserFilters(filter)}
		<Button
			variant="ghost"
			class={buttonSize.default}
			onclick={() => navigate(DEFAULT_USERS_FILTER)}
		>
			{t('admin.users.filters.clear')}
		</Button>
	{/if}
</div>

{#if error}
	<ErrorAlert {error} onRetry={() => void reload()} retryDisabled={loading} />
{:else if !data}
	<LoadingRows rows={6} />
{:else if data.items.length === 0}
	<EmptyState icon={Users} title={t('admin.users.empty.title')} text={t('admin.users.empty.text')}>
		{#snippet action()}
			{#if hasUserFilters(filter) || filter.page > 1}
				<Button
					variant="outline"
					class={buttonSize.sm}
					onclick={() => navigate(DEFAULT_USERS_FILTER)}
				>
					{t('admin.users.filters.clear')}
				</Button>
			{/if}
		{/snippet}
	</EmptyState>
{:else}
	<div class="flex flex-col gap-3" aria-busy={loading}>
		<UsersTable users={data.items} {selfId} {busyId} {onaction} />
		<Pager page={filter.page} total={data.total} onchange={(next) => setFilter({ page: next })} />
	</div>
{/if}

<UserDialog bind:open={dialogOpen} user={dialogUser} {selfId} onCreated={created} onSaved={saved} />

<StepUpDialog
	bind:open={confirmOpen}
	title={confirmTexts.title}
	text={confirmTexts.text}
	confirmLabel={confirmTexts.action}
	destructive={confirmTexts.destructive}
	onConfirm={confirmAction}
>
	{#if confirmKind === 'delete' && confirmUser && confirmUser.sharedLinkCount > 0}
		<StateAlert
			kind="warning"
			title={t('admin.users.confirm.delete.sharedTitle', { count: confirmUser.sharedLinkCount })}
			text={t('admin.users.confirm.delete.sharedText')}
		/>
	{/if}
</StepUpDialog>

<TempPasswordDialog bind:open={tempOpen} username={tempUsername} password={tempPassword} />
