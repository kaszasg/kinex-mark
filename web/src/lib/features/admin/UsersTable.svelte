<script lang="ts" module>
	export type UserAction =
		| 'edit'
		| 'resetPassword'
		| 'resetTwoFactor'
		| 'unlock'
		| 'revokeSessions'
		| 'deactivate'
		| 'activate'
		| 'delete';
</script>

<script lang="ts">
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import Ellipsis from '@lucide/svelte/icons/ellipsis';
	import KeyRound from '@lucide/svelte/icons/key-round';
	import LockOpen from '@lucide/svelte/icons/lock-open';
	import LogOut from '@lucide/svelte/icons/log-out';
	import Pencil from '@lucide/svelte/icons/pencil';
	import ShieldOff from '@lucide/svelte/icons/shield-off';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import UserCheck from '@lucide/svelte/icons/user-check';
	import UserX from '@lucide/svelte/icons/user-x';
	import type { AdminUser } from '$lib/api/types.js';
	import DataTableHead from '$lib/components/app/DataTableHead.svelte';
	import DataTableShell from '$lib/components/app/DataTableShell.svelte';
	import RelativeTime from '$lib/components/app/RelativeTime.svelte';
	import { buttonSize, tableCell } from '$lib/components/app/styles.js';
	import StatusBadge from '$lib/components/brand/StatusBadge.svelte';
	import Button from '$lib/components/app/Button.svelte';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import * as Table from '$lib/components/ui/table/index.js';
	import { fmt, t } from '$lib/i18n/index.js';
	import { cn } from '$lib/utils.js';
	import { isLocked } from './users-query.js';

	/**
	 * Users table of plan 8.7.7 with a row action menu. On the admin's own row deactivation and
	 * deletion are disabled and the menu says where the own account is managed.
	 */
	type Props = {
		users: AdminUser[];
		selfId: number | null;
		/** The id of a user whose direct action (unlock, revoke sessions) is running. */
		busyId?: number | null;
		onaction: (action: UserAction, user: AdminUser) => void;
		class?: string;
	};
	let { users, selfId, busyId = null, onaction, class: className }: Props = $props();

	const num = cn(tableCell, 'text-right font-mono');
</script>

<DataTableShell caption={t('admin.users.table.caption')} class={className}>
	<Table.Header>
		<Table.Row class="hover:bg-transparent">
			<DataTableHead>{t('admin.users.table.username')}</DataTableHead>
			<DataTableHead>{t('admin.users.table.displayName')}</DataTableHead>
			<DataTableHead>{t('admin.users.table.role')}</DataTableHead>
			<DataTableHead>{t('admin.users.table.status')}</DataTableHead>
			<DataTableHead>{t('admin.users.table.twoFactor')}</DataTableHead>
			<DataTableHead numeric>{t('admin.users.table.links')}</DataTableHead>
			<DataTableHead numeric>{t('admin.users.table.sessions')}</DataTableHead>
			<DataTableHead>{t('admin.users.table.lastLogin')}</DataTableHead>
			<DataTableHead class="w-12">
				<span class="sr-only">{t('admin.users.table.actions')}</span>
			</DataTableHead>
		</Table.Row>
	</Table.Header>
	<Table.Body>
		{#each users as user (user.id)}
			{@const self = user.id === selfId}
			{@const locked = isLocked(user.lockedUntil)}
			<Table.Row data-user-id={user.id}>
				<Table.Cell class={cn(tableCell, 'font-mono')}>{user.username}</Table.Cell>
				<Table.Cell class={cn(tableCell, 'max-w-60 truncate')}>
					{user.displayName || fmt.missing}
				</Table.Cell>
				<Table.Cell class={tableCell}>{t(`roles.${user.role}`)}</Table.Cell>
				<Table.Cell class={tableCell}>
					{#if !user.active}
						<StatusBadge status="offline" label={t('status.inactive')} />
					{:else if locked}
						<StatusBadge status="warning" label={t('status.locked')} />
					{:else}
						<StatusBadge status="online" label={t('status.active')} />
					{/if}
				</Table.Cell>
				<Table.Cell class={tableCell}>
					{#if user.totpEnabled}
						<span class="inline-flex items-center gap-1.5">
							<CircleCheck class="size-4 text-success" strokeWidth={1.5} aria-hidden="true" />
							{t('admin.users.table.twoFactorOn')}
						</span>
					{:else}
						<span class="text-muted-foreground">{fmt.missing}</span>
					{/if}
				</Table.Cell>
				<Table.Cell class={num}>{fmt.number(user.linkCount)}</Table.Cell>
				<Table.Cell class={num}>{fmt.number(user.sessionCount)}</Table.Cell>
				<Table.Cell class={cn(tableCell, 'font-mono text-muted-foreground')}>
					{#if user.lastLoginAt}
						<RelativeTime value={user.lastLoginAt} />
					{:else}
						{fmt.missing}
					{/if}
				</Table.Cell>
				<Table.Cell class={cn(tableCell, 'py-1.5 text-right')}>
					<DropdownMenu.Root>
						<DropdownMenu.Trigger disabled={busyId === user.id}>
							{#snippet child({ props })}
								<Button
									{...props}
									variant="ghost"
									size="icon"
									class={buttonSize.iconSm}
									aria-label={t('admin.users.actions.menu', { username: user.username })}
								>
									<Ellipsis class="size-4" strokeWidth={1.5} aria-hidden="true" />
								</Button>
							{/snippet}
						</DropdownMenu.Trigger>
						<DropdownMenu.Content align="end" class="min-w-56">
							<DropdownMenu.Item onSelect={() => onaction('edit', user)}>
								<Pencil class="size-4" strokeWidth={1.5} aria-hidden="true" />
								{t('admin.users.actions.edit')}
							</DropdownMenu.Item>
							<DropdownMenu.Item onSelect={() => onaction('resetPassword', user)}>
								<KeyRound class="size-4" strokeWidth={1.5} aria-hidden="true" />
								{t('admin.users.actions.resetPassword')}
							</DropdownMenu.Item>
							{#if user.totpEnabled}
								<DropdownMenu.Item onSelect={() => onaction('resetTwoFactor', user)}>
									<ShieldOff class="size-4" strokeWidth={1.5} aria-hidden="true" />
									{t('admin.users.actions.resetTwoFactor')}
								</DropdownMenu.Item>
							{/if}
							{#if locked}
								<DropdownMenu.Item onSelect={() => onaction('unlock', user)}>
									<LockOpen class="size-4" strokeWidth={1.5} aria-hidden="true" />
									{t('admin.users.actions.unlock')}
								</DropdownMenu.Item>
							{/if}
							<DropdownMenu.Item onSelect={() => onaction('revokeSessions', user)}>
								<LogOut class="size-4" strokeWidth={1.5} aria-hidden="true" />
								{t('admin.users.actions.revokeSessions')}
							</DropdownMenu.Item>
							<DropdownMenu.Separator />
							{#if user.active}
								<DropdownMenu.Item
									disabled={self}
									title={self ? t('admin.users.actions.selfHint') : undefined}
									onSelect={() => onaction('deactivate', user)}
								>
									<UserX class="size-4" strokeWidth={1.5} aria-hidden="true" />
									{t('admin.users.actions.deactivate')}
								</DropdownMenu.Item>
							{:else}
								<DropdownMenu.Item onSelect={() => onaction('activate', user)}>
									<UserCheck class="size-4" strokeWidth={1.5} aria-hidden="true" />
									{t('admin.users.actions.activate')}
								</DropdownMenu.Item>
							{/if}
							<DropdownMenu.Item
								variant="destructive"
								disabled={self}
								title={self ? t('admin.users.actions.selfHint') : undefined}
								onSelect={() => onaction('delete', user)}
							>
								<Trash2 class="size-4" strokeWidth={1.5} aria-hidden="true" />
								{t('admin.users.actions.delete')}
							</DropdownMenu.Item>
							{#if self}
								<DropdownMenu.Separator />
								<p class="max-w-56 px-2 py-1.5 text-[12px] leading-4 text-muted-foreground">
									{t('admin.users.actions.selfHint')}
								</p>
							{/if}
						</DropdownMenu.Content>
					</DropdownMenu.Root>
				</Table.Cell>
			</Table.Row>
		{/each}
	</Table.Body>
</DataTableShell>
