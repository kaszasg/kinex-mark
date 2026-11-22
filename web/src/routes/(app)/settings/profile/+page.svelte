<script lang="ts">
	// Settings > Profile (plan 8.7.6); the settings layout draws the heading and the tabs.
	import PageTitle from '$lib/components/app/PageTitle.svelte';
	import ProfileForm from '$lib/features/settings/profile/ProfileForm.svelte';
	import { t } from '$lib/i18n/index.js';
</script>

<PageTitle title={t('settings.pageTitle', { section: t('nav.settingsTabs.profile') })} />
<ProfileForm />
