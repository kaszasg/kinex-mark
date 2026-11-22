<script lang="ts">
	import Button from './Button.svelte';
	import Panel from '$lib/components/brand/Panel.svelte';
	import { t, type MessageKey } from '$lib/i18n/index.js';
	import PageTitle from './PageTitle.svelte';
	import { buttonSize } from './styles.js';

	/** Error page body of plan 8.7.11: mono status code, title, text and a way back to the links. */
	type Props = { status: number };
	let { status }: Props = $props();

	const keys = $derived<{ title: MessageKey; text: MessageKey }>(
		status === 404
			? { title: 'nav.errorPage.notFound.title', text: 'nav.errorPage.notFound.text' }
			: status === 403
				? { title: 'nav.errorPage.forbidden.title', text: 'nav.errorPage.forbidden.text' }
				: { title: 'errors.internal_error.title', text: 'errors.internal_error.text' }
	);
</script>

<PageTitle title={t(keys.title)} />

<section class="flex w-full max-w-[var(--layout-prose)] flex-col items-start gap-4 py-6">
	<Panel size="sm">
		<span class="block px-3 py-1.5 font-mono text-[13px] leading-5 text-muted-foreground">
			{status}
		</span>
	</Panel>
	<div class="flex flex-col gap-2">
		<h1 class="heading-1 text-balance">{t(keys.title)}</h1>
		<p class="body text-muted-foreground">{t(keys.text)}</p>
	</div>
	<Button href="/" class={buttonSize.default}>{t('nav.errorPage.notFound.action')}</Button>
</section>
