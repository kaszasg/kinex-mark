<script lang="ts">
	import PageTitle from '$lib/components/app/PageTitle.svelte';
	import PasswordChangeForm from '$lib/features/auth/PasswordChangeForm.svelte';
	import { t } from '$lib/i18n/index.js';
</script>

<PageTitle title={t('auth.passwordChange.pageTitle')} />

<section class="flex flex-col gap-6" aria-labelledby="password-gate-title">
	<div class="flex flex-col gap-1">
		<h1 id="password-gate-title" class="heading-2 text-foreground">
			{t('auth.passwordChange.title')}
		</h1>
		<p class="text-[13px] leading-5 text-muted-foreground">{t('auth.passwordChange.text')}</p>
	</div>
	<PasswordChangeForm mode="gate" />
</section>
