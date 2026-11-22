<script lang="ts">
	import '../app.css';
	import '../kinex-mark.css';
	import { ModeWatcher } from 'mode-watcher';
	import AppToaster from '$lib/components/app/AppToaster.svelte';
	import { i18n } from '$lib/i18n/index.js';
	let { children } = $props();

	// <html lang> always follows the active locale (plan 8.1.7).
	$effect(() => {
		document.documentElement.lang = i18n.locale;
	});
</script>

<ModeWatcher defaultMode="dark" lightClassNames={['light']} disableHeadScriptInjection />
<AppToaster />
{@render children()}
