<script lang="ts">
	// Settings > Security (plan 8.7.6): password, two-factor sign-in and sessions.
	import PageTitle from '$lib/components/app/PageTitle.svelte';
	import PasswordCard from '$lib/features/settings/security/PasswordCard.svelte';
	import SessionsCard from '$lib/features/settings/security/SessionsCard.svelte';
	import TwoFactorCard from '$lib/features/settings/security/TwoFactorCard.svelte';
	import { t } from '$lib/i18n/index.js';
</script>

<PageTitle title={t('settings.pageTitle', { section: t('nav.settingsTabs.security') })} />
<PasswordCard />
<TwoFactorCard />
<SessionsCard />
