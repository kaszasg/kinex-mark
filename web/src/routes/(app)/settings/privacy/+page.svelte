<script lang="ts">
	// Settings > Privacy (plan 8.7.6): privacy notice, data export and account deletion.
	import PageTitle from '$lib/components/app/PageTitle.svelte';
	import ConsentHistory from '$lib/features/settings/privacy/ConsentHistory.svelte';
	import DataExportCard from '$lib/features/settings/privacy/DataExportCard.svelte';
	import DeleteAccountDialog from '$lib/features/settings/privacy/DeleteAccountDialog.svelte';
	import { t } from '$lib/i18n/index.js';
</script>

<PageTitle title={t('settings.pageTitle', { section: t('nav.settingsTabs.privacy') })} />
<ConsentHistory />
<DataExportCard />
<DeleteAccountDialog />
