<script lang="ts">
	// Settings > Display (plan 8.7.6); the settings layout draws the heading and the tabs.
	import PageTitle from '$lib/components/app/PageTitle.svelte';
	import DisplaySettings from '$lib/features/settings/display/DisplaySettings.svelte';
	import { t } from '$lib/i18n/index.js';
</script>

<PageTitle title={t('settings.pageTitle', { section: t('nav.settingsTabs.display') })} />
<DisplaySettings />
