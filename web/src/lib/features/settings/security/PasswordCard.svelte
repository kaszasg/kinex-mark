<script lang="ts">
	import PasswordChangeForm from '$lib/features/auth/PasswordChangeForm.svelte';
	import { t } from '$lib/i18n/index.js';
	import SectionCard from './SectionCard.svelte';

	/**
	 * "Password" card of Settings > Security (plan 8.7.6). The form is the shared
	 * PasswordChangeForm of features/auth (F1): it carries the screen's only primary button and
	 * shows the success toast itself in the settings mode, so the card adds none.
	 */
</script>

<SectionCard
	id="security-password"
	title={t('security.password.title')}
	description={t('security.password.text')}
>
	<PasswordChangeForm mode="settings" />
</SectionCard>
