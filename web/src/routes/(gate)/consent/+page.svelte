<script lang="ts">
	import PageTitle from '$lib/components/app/PageTitle.svelte';
	import ConsentForm from '$lib/features/privacy/ConsentForm.svelte';
	import { t } from '$lib/i18n/index.js';
</script>

<PageTitle title={t('privacy.title')} />
<ConsentForm />
