<script lang="ts">
	import { session } from '$lib/state/session.svelte.js';

	/**
	 * The signed-in account's username as a hidden, read-only field of a password form, so that
	 * password managers can link the password to the account (Chromium: "Password forms should
	 * have (optionally hidden) username fields"). It is not focusable, not shown and not in the
	 * accessibility tree, and the form never sends it.
	 */
	let { username }: { username?: string } = $props();
</script>

<input
	type="text"
	name="username"
	autocomplete="username"
	value={username ?? session.me?.username ?? ''}
	readonly
	hidden
	tabindex={-1}
	aria-hidden="true"
	data-account-username
/>
