<script lang="ts" module>
	export type Status = 'online' | 'warning' | 'critical' | 'offline' | 'info';
</script>

<script lang="ts">
	import CircleAlert from '@lucide/svelte/icons/circle-alert';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import Info from '@lucide/svelte/icons/info';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import WifiOff from '@lucide/svelte/icons/wifi-off';
	import { t, type MessageKey } from '$lib/i18n/index.js';

	/** The state is carried by the icon and the word together. The base words come from status.*. */
	type Props = { status: Status; label?: string };
	let { status, label }: Props = $props();

	const map = {
		online: { icon: CircleCheck, text: 'status.online', color: 'text-success' },
		warning: { icon: TriangleAlert, text: 'status.warning', color: 'text-warning' },
		critical: { icon: CircleAlert, text: 'status.critical', color: 'text-destructive' },
		offline: { icon: WifiOff, text: 'status.offline', color: 'text-muted-foreground' },
		info: { icon: Info, text: 'status.info', color: 'text-info' }
	} as const satisfies Record<Status, { icon: unknown; text: MessageKey; color: string }>;
	const item = $derived(map[status]);
</script>

<span
	class="inline-flex h-[22px] items-center gap-1.5 rounded-sm border px-2 text-xs font-medium whitespace-nowrap"
>
	<item.icon class="size-3 {item.color}" strokeWidth={1.5} aria-hidden="true" />
	{label ?? t(item.text)}
</span>
