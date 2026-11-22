<script lang="ts">
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import { Pagination as PaginationPrimitive } from 'bits-ui';
	import { buttonVariants } from '$lib/components/ui/button/index.js';
	import * as Pagination from '$lib/components/ui/pagination/index.js';
	import { buttonSize, buttonTone } from '$lib/components/app/styles.js';
	import { fmt, t } from '$lib/i18n/index.js';
	import { cn } from '$lib/utils.js';
	import { ADMIN_PAGE_SIZE, pageCount, pageRange } from './paging.js';

	/**
	 * Page navigation of the admin tables (plan 8.7.7): the shown range "{from}-{to} / {total}"
	 * in mono and shadcn Pagination with translated labels (the generated previous/next parts
	 * carry English text, so they are rebuilt here on the bits-ui primitives).
	 */
	type Props = {
		page: number;
		total: number;
		size?: number;
		onchange: (page: number) => void;
		class?: string;
	};
	let { page, total, size = ADMIN_PAGE_SIZE, onchange, class: className }: Props = $props();

	const range = $derived(pageRange(page, total, size));
	const pages = $derived(pageCount(total, size));
</script>

<div class={cn('flex flex-wrap items-center justify-between gap-x-6 gap-y-2', className)}>
	<p class="font-mono text-[13px] leading-5 text-muted-foreground" aria-live="polite">
		{t('admin.pager.range', {
			from: fmt.number(range.from),
			to: fmt.number(range.to),
			total: fmt.number(total)
		})}
	</p>
	{#if pages > 1}
		<Pagination.Root
			count={total}
			perPage={size}
			page={Math.min(page, pages)}
			onPageChange={(next: number) => onchange(next)}
			aria-label={t('admin.pager.label')}
			class="mx-0 w-auto"
		>
			{#snippet children({ pages: items, currentPage })}
				<Pagination.Content class="gap-1">
					<Pagination.Item>
						<PaginationPrimitive.PrevButton
							aria-label={t('admin.pager.previous')}
							class={cn(buttonVariants({ variant: 'ghost' }), buttonTone.ghost, buttonSize.iconSm)}
						>
							<ChevronLeft class="size-4" strokeWidth={1.5} aria-hidden="true" />
						</PaginationPrimitive.PrevButton>
					</Pagination.Item>
					{#each items as item (item.key)}
						<Pagination.Item>
							{#if item.type === 'ellipsis'}
								<span
									class="inline-flex size-7 items-center justify-center font-mono text-[13px] text-muted-foreground"
									aria-hidden="true"
								>
									…
								</span>
							{:else}
								<PaginationPrimitive.Page page={item}>
									{#snippet child({ props })}
										<!-- The primitive names pages in English; the label is replaced after the spread. -->
										<button
											{...props}
											aria-label={t('admin.pager.page', { page: item.value })}
											aria-current={currentPage === item.value ? 'page' : undefined}
											class={cn(
												buttonVariants({
													variant: currentPage === item.value ? 'outline' : 'ghost',
													size: 'icon'
												}),
												buttonTone[currentPage === item.value ? 'outline' : 'ghost'],
												buttonSize.iconSm,
												'font-mono text-[13px]'
											)}
										>
											{item.value}
										</button>
									{/snippet}
								</PaginationPrimitive.Page>
							{/if}
						</Pagination.Item>
					{/each}
					<Pagination.Item>
						<PaginationPrimitive.NextButton
							aria-label={t('admin.pager.next')}
							class={cn(buttonVariants({ variant: 'ghost' }), buttonTone.ghost, buttonSize.iconSm)}
						>
							<ChevronRight class="size-4" strokeWidth={1.5} aria-hidden="true" />
						</PaginationPrimitive.NextButton>
					</Pagination.Item>
				</Pagination.Content>
			{/snippet}
		</Pagination.Root>
	{/if}
</div>
