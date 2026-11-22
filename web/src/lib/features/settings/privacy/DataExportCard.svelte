<script lang="ts">
	import FileDown from '@lucide/svelte/icons/file-down';
	import { toast } from 'svelte-sonner';
	import * as meApi from '$lib/api/me.js';
	import ErrorAlert from '$lib/components/app/ErrorAlert.svelte';
	import { buttonSize } from '$lib/components/app/styles.js';
	import Button from '$lib/components/app/Button.svelte';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import SectionCard from '$lib/features/settings/security/SectionCard.svelte';
	import { t } from '$lib/i18n/index.js';
	import { saveBlob } from '$lib/utils/download.js';

	/**
	 * "Download your data" card of plan 8.7.6 (GDPR access and portability): what the export
	 * contains, and GET /me/export saved under the server's file name.
	 */
	const items = [
		'security.privacy.export.items.account',
		'security.privacy.export.items.preferences',
		'security.privacy.export.items.sessions',
		'security.privacy.export.items.tags',
		'security.privacy.export.items.links',
		'security.privacy.export.items.shared',
		'security.privacy.export.items.audit',
		'security.privacy.export.items.lock'
	] as const;

	let busy = $state(false);
	let error = $state<unknown>(null);

	async function download(): Promise<void> {
		if (busy) return;
		busy = true;
		error = null;
		try {
			const { blob, filename } = await meApi.exportMyData();
			saveBlob(blob, filename);
			toast.success(t('security.privacy.export.done'));
		} catch (err) {
			error = err;
		} finally {
			busy = false;
		}
	}
</script>

<SectionCard
	id="privacy-export"
	title={t('security.privacy.export.title')}
	description={t('security.privacy.export.text')}
>
	<ul class="body-sm flex list-disc flex-col gap-0.5 pl-5 marker:text-muted-foreground">
		{#each items as key (key)}
			<li>{t(key)}</li>
		{/each}
	</ul>
	<p class="body-sm text-muted-foreground">{t('security.privacy.export.excluded')}</p>
	{#if error}
		<ErrorAlert {error} />
	{/if}
	<div>
		<Button
			variant="outline"
			class={buttonSize.default}
			disabled={busy}
			aria-busy={busy}
			onclick={download}
		>
			{#if busy}
				<Spinner class="size-4" aria-label={t('common.states.loading')} />
			{:else}
				<FileDown class="size-4" strokeWidth={1.5} aria-hidden="true" />
			{/if}
			{t('security.privacy.export.submit')}
		</Button>
	</div>
</SectionCard>
