<script lang="ts">
	// Test double of the F1 PrivacyNotice (same props); the component specs use it via vi.mock.
	import type { PublicConfig } from '$lib/api/types.js';

	type Props = { config: PublicConfig['privacy']; compact?: boolean };
	let { config, compact = false }: Props = $props();
</script>

<div
	data-testid="privacy-notice"
	data-compact={compact ? 'true' : 'false'}
	data-version={config.noticeVersion}
></div>
