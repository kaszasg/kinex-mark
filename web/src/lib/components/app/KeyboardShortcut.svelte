<script lang="ts">
	import * as Kbd from '$lib/components/ui/kbd/index.js';
	import { keyLabel, modifierKeyLabel } from '$lib/utils/keyboard.js';

	/**
	 * Key combination of plan 8.4. `mod` prepends the platform's primary modifier (Ctrl or the
	 * Command sign); `keys` are the remaining keys in order. Named keys (Enter, Delete, Esc) and
	 * the modifier are printed in the active language.
	 */
	type Props = { keys: string[]; mod?: boolean; class?: string };
	let { keys, mod = false, class: className }: Props = $props();

	const all = $derived(mod ? [modifierKeyLabel(), ...keys.map(keyLabel)] : keys.map(keyLabel));
</script>

<Kbd.Group class={className}>
	{#each all as key, index (index)}
		<Kbd.Root class="font-mono text-[11px]">{key}</Kbd.Root>
	{/each}
</Kbd.Group>
