<script lang="ts">
	import type { Snippet } from 'svelte';
	import * as Table from '$lib/components/ui/table/index.js';
	import { cn } from '$lib/utils.js';
	import { tableHead } from './styles.js';

	/** Header cell of GRAPHICS.md 10.7; `numeric` right-aligns a number column. */
	type Props = {
		numeric?: boolean;
		sort?: 'ascending' | 'descending' | 'none';
		class?: string;
		children?: Snippet;
	};
	let { numeric = false, sort, class: className, children }: Props = $props();
</script>

<Table.Head
	scope="col"
	aria-sort={sort}
	class={cn(tableHead, numeric ? 'text-right' : 'text-left', className)}
>
	{@render children?.()}
</Table.Head>
