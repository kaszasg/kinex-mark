<script lang="ts">
	// Test double of the F1 PasswordChangeForm (same props); the component specs use it via vi.mock.
	type Props = { mode: 'gate' | 'settings'; onSuccess?: () => void };
	let { mode, onSuccess }: Props = $props();
</script>

<button
	type="button"
	data-testid="password-change-form"
	data-mode={mode}
	aria-label={mode}
	onclick={() => onSuccess?.()}
></button>
