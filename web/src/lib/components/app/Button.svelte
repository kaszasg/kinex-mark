<script lang="ts">
	import { Button as GeneratedButton, type ButtonProps } from '$lib/components/ui/button/index.js';
	import { cn } from '$lib/utils.js';
	import { buttonTone } from './styles.js';

	/**
	 * The application's button: the generated shadcn-svelte Button (kept unedited, plan 8.1.5) with
	 * the per-variant GRAPHICS.md fixes of `buttonTone` (radius-md, hover, transparent outline in
	 * the dark theme). Sizes still come from `buttonSize` at the call site.
	 */
	let {
		ref = $bindable(null),
		variant = 'default',
		class: className,
		...rest
	}: ButtonProps = $props();
</script>

<GeneratedButton
	bind:ref
	{variant}
	class={cn(buttonTone[variant ?? 'default'], className)}
	{...rest}
/>
