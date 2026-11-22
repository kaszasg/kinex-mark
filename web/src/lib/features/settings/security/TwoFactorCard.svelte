<script lang="ts" module>
	/** At or below this many unused recovery codes the card warns (plan 8.7.6). */
	export const LOW_RECOVERY_CODES = 3;
</script>

<script lang="ts">
	import KeyRound from '@lucide/svelte/icons/key-round';
	import ShieldCheck from '@lucide/svelte/icons/shield-check';
	import ShieldOff from '@lucide/svelte/icons/shield-off';
	import StateAlert from '$lib/components/app/StateAlert.svelte';
	import { buttonSize } from '$lib/components/app/styles.js';
	import StatusBadge from '$lib/components/brand/StatusBadge.svelte';
	import Button from '$lib/components/app/Button.svelte';
	import { fmt, t } from '$lib/i18n/index.js';
	import { session } from '$lib/state/session.svelte.js';
	import SectionCard from './SectionCard.svelte';
	import TwoFactorConfirmDialog, {
		type TwoFactorConfirmMode
	} from './TwoFactorConfirmDialog.svelte';
	import TwoFactorEnrollDialog from './TwoFactorEnrollDialog.svelte';

	/**
	 * "Two-factor sign-in" card of plan 8.7.6: status badge, unused recovery codes with a
	 * warning at three or fewer, and the enrol, new codes and turn-off actions (all outline).
	 */
	const enabled = $derived(session.me?.totpEnabled ?? false);
	const remaining = $derived(session.me?.recoveryCodesRemaining ?? 0);

	let enrollOpen = $state(false);
	let confirmOpen = $state(false);
	let confirmMode = $state<TwoFactorConfirmMode>('disable');

	function openConfirm(mode: TwoFactorConfirmMode): void {
		confirmMode = mode;
		confirmOpen = true;
	}
</script>

<SectionCard
	id="security-two-factor"
	title={t('security.twoFactor.title')}
	description={t('security.twoFactor.text')}
>
	{#snippet aside()}
		<StatusBadge
			status={enabled ? 'online' : 'offline'}
			label={enabled ? t('status.enabled') : t('status.disabled')}
		/>
	{/snippet}

	{#if enabled}
		<dl class="flex items-baseline justify-between gap-4 border-b pb-3">
			<dt class="body-sm text-muted-foreground">{t('security.twoFactor.remaining')}</dt>
			<dd class="data font-mono text-[13px] leading-5" data-testid="recovery-remaining">
				{fmt.number(remaining)}
			</dd>
		</dl>
		{#if remaining <= LOW_RECOVERY_CODES}
			<StateAlert
				kind="warning"
				title={t('security.twoFactor.lowCodes.title')}
				text={t('security.twoFactor.lowCodes.text')}
			/>
		{/if}
		<div class="flex flex-wrap gap-2">
			<Button
				variant="outline"
				class={buttonSize.default}
				onclick={() => openConfirm('regenerate')}
			>
				<KeyRound class="size-4" strokeWidth={1.5} aria-hidden="true" />
				{t('security.twoFactor.regenerate')}
			</Button>
			<Button variant="outline" class={buttonSize.default} onclick={() => openConfirm('disable')}>
				<ShieldOff class="size-4" strokeWidth={1.5} aria-hidden="true" />
				{t('security.twoFactor.disable')}
			</Button>
		</div>
	{:else}
		<div class="flex flex-wrap gap-2">
			<Button variant="outline" class={buttonSize.default} onclick={() => (enrollOpen = true)}>
				<ShieldCheck class="size-4" strokeWidth={1.5} aria-hidden="true" />
				{t('security.twoFactor.enable')}
			</Button>
		</div>
	{/if}
</SectionCard>

<TwoFactorEnrollDialog bind:open={enrollOpen} />
<TwoFactorConfirmDialog bind:open={confirmOpen} mode={confirmMode} />
