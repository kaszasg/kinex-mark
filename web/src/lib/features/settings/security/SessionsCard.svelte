<script lang="ts" module>
	import type { Session } from '$lib/api/types.js';
	import { describeUserAgent, rawUserAgent } from '$lib/utils/user-agent.js';

	/** Device text of a session: "Chrome, Windows", else the raw user agent in mono, else null. */
	export function sessionDevice(item: Pick<Session, 'userAgent'>): {
		text: string | null;
		mono: boolean;
	} {
		const described = describeUserAgent(item.userAgent);
		if (described) return { text: described, mono: false };
		const raw = rawUserAgent(item.userAgent).trim();
		return raw ? { text: raw, mono: true } : { text: null, mono: false };
	}

	/** The current session first, the rest in server order (newest first). */
	export function orderSessions(items: Session[]): Session[] {
		return [...items.filter((s) => s.current), ...items.filter((s) => !s.current)];
	}
</script>

<script lang="ts">
	import LogOut from '@lucide/svelte/icons/log-out';
	import MonitorSmartphone from '@lucide/svelte/icons/monitor-smartphone';
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { goto } from '$app/navigation';
	import { ApiError } from '$lib/api/errors.js';
	import * as meApi from '$lib/api/me.js';
	import ConfirmDialog from '$lib/components/app/ConfirmDialog.svelte';
	import EmptyState from '$lib/components/app/EmptyState.svelte';
	import ErrorAlert from '$lib/components/app/ErrorAlert.svelte';
	import LoadingRows from '$lib/components/app/LoadingRows.svelte';
	import RelativeTime from '$lib/components/app/RelativeTime.svelte';
	import { buttonSize } from '$lib/components/app/styles.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import Button from '$lib/components/app/Button.svelte';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { fmt, t } from '$lib/i18n/index.js';
	import { session } from '$lib/state/session.svelte.js';
	import { cn } from '$lib/utils.js';
	import SectionCard from './SectionCard.svelte';
	import { toastError } from '$lib/utils/toast-error.js';

	/**
	 * "Sessions" card of plan 8.7.6: every live session with device, IP and times; the current
	 * one is marked "This device". Ending another session is immediate; ending the current one
	 * asks first and then goes to /login. The head action ends every other session.
	 */
	let status = $state<'loading' | 'ready' | 'error'>('loading');
	let items = $state<Session[]>([]);
	let error = $state<unknown>(null);
	let revoking = $state<number | null>(null);
	let revokingOthers = $state(false);
	let confirmOpen = $state(false);
	let pendingCurrent = $state<Session | null>(null);

	const otherCount = $derived(items.filter((s) => !s.current).length);

	function deviceText(item: Session): string {
		return sessionDevice(item).text ?? t('security.sessions.unknownDevice');
	}

	async function load(): Promise<void> {
		status = 'loading';
		error = null;
		try {
			items = orderSessions((await meApi.listSessions()).items);
			status = 'ready';
		} catch (err) {
			error = err;
			status = 'error';
		}
	}

	onMount(() => {
		void load();
	});

	async function revoke(item: Session): Promise<void> {
		if (item.current) {
			pendingCurrent = item;
			confirmOpen = true;
			return;
		}
		revoking = item.id;
		try {
			await meApi.revokeSession(item.id);
			items = items.filter((s) => s.id !== item.id);
			toast.success(t('security.sessions.revoked', { device: deviceText(item) }));
		} catch (err) {
			if (err instanceof ApiError && err.code === 'not_found') {
				items = items.filter((s) => s.id !== item.id);
			}
			toastError(err);
		} finally {
			revoking = null;
		}
	}

	async function revokeCurrent(): Promise<void> {
		if (!pendingCurrent) return;
		await meApi.revokeSession(pendingCurrent.id);
		session.clear('logout');
		await goto('/login');
	}

	async function revokeOthers(): Promise<void> {
		revokingOthers = true;
		try {
			const { revoked } = await meApi.revokeOtherSessions();
			items = items.filter((s) => s.current);
			toast.success(t('security.sessions.revokedOthers', { count: revoked }));
		} catch (err) {
			toastError(err);
		} finally {
			revokingOthers = false;
		}
	}
</script>

<SectionCard
	id="security-sessions"
	title={t('security.sessions.title')}
	description={t('security.sessions.text')}
>
	{#snippet aside()}
		<Button
			variant="outline"
			class={buttonSize.sm}
			disabled={status !== 'ready' || otherCount === 0 || revokingOthers}
			aria-busy={revokingOthers}
			onclick={revokeOthers}
		>
			{#if revokingOthers}<Spinner class="size-4" aria-label={t('common.states.loading')} />{/if}
			{t('security.sessions.revokeOthers')}
		</Button>
	{/snippet}

	{#if status === 'loading'}
		<LoadingRows rows={2} rowClass="h-16" />
	{:else if status === 'error'}
		<ErrorAlert {error} onRetry={load} />
	{:else if items.length === 0}
		<EmptyState
			icon={MonitorSmartphone}
			title={t('security.sessions.empty.title')}
			text={t('security.sessions.empty.text')}
		/>
	{:else}
		<ul aria-label={t('security.sessions.listLabel')} class="-my-1 flex flex-col">
			{#each items as item (item.id)}
				{@const device = sessionDevice(item)}
				<li
					class="flex items-start gap-3 border-b py-3 first:pt-1 last:border-b-0 last:pb-1"
					data-current={item.current ? 'true' : undefined}
				>
					<div class="flex min-w-0 flex-1 flex-col gap-2">
						<div class="flex min-w-0 flex-wrap items-center gap-2">
							<span
								class={cn(
									'min-w-0 truncate text-[13px] leading-5',
									device.mono ? 'font-mono' : 'font-medium'
								)}
							>
								{device.text ?? t('security.sessions.unknownDevice')}
							</span>
							{#if item.current}
								<Badge variant="outline" class="h-5 rounded-sm px-1.5 text-[11px] leading-4">
									{t('security.sessions.current')}
								</Badge>
							{/if}
						</div>
						<dl class="grid grid-cols-2 gap-x-4 gap-y-1.5 sm:grid-cols-4">
							<div class="flex min-w-0 flex-col">
								<dt class="text-[11px] leading-4 text-muted-foreground">
									{t('security.sessions.ip')}
								</dt>
								<dd class="truncate font-mono text-[12px] leading-5" title={item.ip}>
									{item.ip || fmt.missing}
								</dd>
							</div>
							<div class="flex min-w-0 flex-col">
								<dt class="text-[11px] leading-4 text-muted-foreground">
									{t('security.sessions.created')}
								</dt>
								<dd class="font-mono text-[12px] leading-5">{fmt.dateTime(item.createdAt)}</dd>
							</div>
							<div class="flex min-w-0 flex-col">
								<dt class="text-[11px] leading-4 text-muted-foreground">
									{t('security.sessions.lastSeen')}
								</dt>
								<dd class="font-mono text-[12px] leading-5">
									<RelativeTime value={item.lastSeenAt} />
								</dd>
							</div>
							<div class="flex min-w-0 flex-col">
								<dt class="text-[11px] leading-4 text-muted-foreground">
									{t('security.sessions.expires')}
								</dt>
								<dd class="font-mono text-[12px] leading-5">{fmt.dateTime(item.expiresAt)}</dd>
							</div>
						</dl>
					</div>
					<Button
						variant="ghost"
						size="icon"
						class={buttonSize.icon}
						aria-label={t('security.sessions.revoke', { device: deviceText(item) })}
						disabled={revoking !== null}
						onclick={() => revoke(item)}
					>
						{#if revoking === item.id}
							<Spinner class="size-4" aria-label={t('common.states.loading')} />
						{:else}
							<LogOut class="size-4" strokeWidth={1.5} aria-hidden="true" />
						{/if}
					</Button>
				</li>
			{/each}
		</ul>
	{/if}
</SectionCard>

<ConfirmDialog
	bind:open={confirmOpen}
	title={t('security.sessions.revokeCurrent.title')}
	text={t('security.sessions.revokeCurrent.text')}
	confirmLabel={t('security.sessions.revokeCurrent.submit')}
	destructive
	onConfirm={revokeCurrent}
	onCancel={() => (pendingCurrent = null)}
/>
