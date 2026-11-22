<script lang="ts">
	import PageTitle from '$lib/components/app/PageTitle.svelte';
	import AuditScreen from '$lib/features/admin/AuditScreen.svelte';
	import { t } from '$lib/i18n/index.js';
</script>

<PageTitle title={t('audit.pageTitle')} />
<AuditScreen />
