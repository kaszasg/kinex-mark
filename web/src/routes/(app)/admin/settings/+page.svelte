<script lang="ts">
	import PageTitle from '$lib/components/app/PageTitle.svelte';
	import SettingsForm from '$lib/features/admin/SettingsForm.svelte';
	import { t } from '$lib/i18n/index.js';
</script>

<PageTitle title={t('admin.settings.pageTitle')} />
<SettingsForm />
