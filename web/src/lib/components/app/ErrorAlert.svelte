<script lang="ts">
	import { ApiError } from '$lib/api/errors.js';
	import { errorText, errorTitle, t } from '$lib/i18n/index.js';
	import StateAlert from './StateAlert.svelte';

	/** ApiError -> critical StateAlert: errors.<code>.title and errors.<code>.text, optional retry. */
	type Props = { error: unknown; onRetry?: () => void; retryDisabled?: boolean; class?: string };
	let { error, onRetry, retryDisabled = false, class: className }: Props = $props();

	const apiError = $derived(error instanceof ApiError ? error : new ApiError(0, 'internal_error'));
</script>

<StateAlert
	kind="critical"
	title={errorTitle(apiError)}
	text={errorText(apiError)}
	actionLabel={onRetry ? t('common.actions.retry') : undefined}
	onAction={onRetry}
	actionDisabled={retryDisabled}
	class={className}
/>
