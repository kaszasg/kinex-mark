<script lang="ts">
	import { t } from '$lib/i18n/index.js';

	/** Document title of plan 8.1.7: "<page> - Kinex Mark" (en dash), or the app name alone. */
	let { title }: { title?: string | null } = $props();

	const full = $derived(
		title ? `${title} – ${t('common.pageTitleSuffix')}` : t('common.pageTitleSuffix')
	);
</script>

<svelte:head>
	<title>{full}</title>
</svelte:head>
