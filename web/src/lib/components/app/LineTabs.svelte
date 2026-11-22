<script lang="ts" module>
	export type LineTab = { href: string; label: string };
</script>

<script lang="ts">
	import { cn } from '$lib/utils.js';

	/**
	 * Line tabs of GRAPHICS.md 10.18 as page links (the settings tabs are routes, so they are
	 * links with aria-current, not a tablist). Tabs.Root views use the classes in styles.ts.
	 */
	type Props = { items: LineTab[]; current: string; label: string; class?: string };
	let { items, current, label, class: className }: Props = $props();
</script>

<nav aria-label={label} class={cn('w-full overflow-x-auto', className)}>
	<ul class="flex min-w-max gap-6 border-b">
		{#each items as item (item.href)}
			{@const active = current === item.href || current.startsWith(`${item.href}/`)}
			<li class="flex">
				<a
					href={item.href}
					aria-current={active ? 'page' : undefined}
					class={cn(
						'relative pt-2 pb-2.5 text-[13px] leading-5 font-medium text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring',
						active &&
							'text-foreground after:absolute after:inset-x-0 after:-bottom-px after:h-0.5 after:bg-primary'
					)}
				>
					{item.label}
				</a>
			</li>
		{/each}
	</ul>
</nav>
