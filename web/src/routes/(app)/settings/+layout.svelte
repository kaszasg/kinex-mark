<script lang="ts">
	import { page } from '$app/state';
	import LineTabs from '$lib/components/app/LineTabs.svelte';
	import PageHeader from '$lib/components/app/PageHeader.svelte';
	import { t } from '$lib/i18n/index.js';

	/** Settings frame of plan 8.7.6: heading and four line tabs as links; content at prose width. */
	let { children } = $props();

	const tabs = $derived([
		{ href: '/settings/profile', label: t('nav.settingsTabs.profile') },
		{ href: '/settings/security', label: t('nav.settingsTabs.security') },
		{ href: '/settings/display', label: t('nav.settingsTabs.display') },
		{ href: '/settings/privacy', label: t('nav.settingsTabs.privacy') }
	]);
</script>

<PageHeader title={t('nav.settings')} />
<LineTabs items={tabs} current={page.url.pathname} label={t('nav.settingsTabs.label')} />
<div class="flex w-full max-w-[var(--layout-prose)] flex-col gap-4 pt-2">
	{@render children()}
</div>
