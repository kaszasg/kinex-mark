<script lang="ts" module>
	export type Actor = { id: number; username: string; displayName: string };
</script>

<script lang="ts">
	import Check from '@lucide/svelte/icons/check';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import { inputSize } from '$lib/components/app/styles.js';
	import * as Command from '$lib/components/ui/command/index.js';
	import * as Popover from '$lib/components/ui/popover/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { t } from '$lib/i18n/index.js';
	import { cn } from '$lib/utils.js';

	/**
	 * Actor filter of the audit log (plan 8.7.9): Popover + Command over the user list, with an
	 * "All" row. `actors` is null while the list loads; `failed` says it could not be loaded.
	 */
	type Props = {
		id: string;
		value: number | null;
		actors: Actor[] | null;
		failed?: boolean;
		onchange: (id: number | null) => void;
	};
	let { id, value, actors, failed = false, onchange }: Props = $props();

	let open = $state(false);
	let search = $state('');

	const selected = $derived(actors?.find((actor) => actor.id === value) ?? null);
	const needle = $derived(search.trim().toLowerCase());
	const matches = $derived(
		(actors ?? []).filter(
			(actor) =>
				!needle ||
				actor.username.toLowerCase().includes(needle) ||
				actor.displayName.toLowerCase().includes(needle)
		)
	);
	const label = $derived(
		value === null
			? t('audit.filters.all')
			: (selected?.username ?? t('audit.filters.actorUnknown', { id: String(value) }))
	);

	function choose(next: number | null): void {
		open = false;
		search = '';
		if (next !== value) onchange(next);
	}
</script>

<Popover.Root bind:open>
	<Popover.Trigger {id}>
		{#snippet child({ props })}
			<button
				{...props}
				type="button"
				class={cn(
					inputSize,
					'flex w-full min-w-0 items-center justify-between gap-2 rounded-md border border-input bg-transparent text-left outline-none'
				)}
			>
				<span class={cn('truncate', value !== null && selected && 'font-mono')}>{label}</span>
				<ChevronDown
					class="size-4 shrink-0 text-muted-foreground"
					strokeWidth={1.5}
					aria-hidden="true"
				/>
			</button>
		{/snippet}
	</Popover.Trigger>
	<Popover.Content class="w-72 p-0" align="start">
		<Command.Root shouldFilter={false} label={t('audit.filters.actorSearch')}>
			<Command.Input bind:value={search} placeholder={t('audit.filters.actorSearch')} />
			<Command.List aria-label={t('audit.filters.actor')}>
				{#if failed}
					<p class="px-3 py-4 text-[13px] leading-5 text-muted-foreground">
						{t('audit.filters.actorUnavailable')}
					</p>
				{:else if actors === null}
					<div class="flex justify-center py-4">
						<Spinner class="size-4" aria-label={t('common.states.loading')} />
					</div>
				{:else}
					<Command.Group>
						{#if !needle}
							<Command.Item value="__all" onSelect={() => choose(null)}>
								<Check
									class={cn('size-4', value !== null && 'invisible')}
									strokeWidth={1.5}
									aria-hidden="true"
								/>
								{t('audit.filters.all')}
							</Command.Item>
						{/if}
						{#each matches as actor (actor.id)}
							<Command.Item value={String(actor.id)} onSelect={() => choose(actor.id)}>
								<Check
									class={cn('size-4', value !== actor.id && 'invisible')}
									strokeWidth={1.5}
									aria-hidden="true"
								/>
								<span class="font-mono">{actor.username}</span>
								{#if actor.displayName}
									<span class="truncate text-muted-foreground">{actor.displayName}</span>
								{/if}
							</Command.Item>
						{/each}
					</Command.Group>
					{#if matches.length === 0}
						<p class="px-3 py-4 text-center text-[13px] leading-5 text-muted-foreground">
							{t('audit.filters.actorEmpty')}
						</p>
					{/if}
				{/if}
			</Command.List>
		</Command.Root>
	</Popover.Content>
</Popover.Root>
