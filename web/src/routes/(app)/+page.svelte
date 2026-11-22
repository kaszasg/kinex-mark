<script lang="ts">
	import PageTitle from '$lib/components/app/PageTitle.svelte';
	import LinksPage from '$lib/features/links/LinksPage.svelte';
	import { t } from '$lib/i18n/index.js';
</script>

<PageTitle title={t('links.pageTitle')} />
<LinksPage />
