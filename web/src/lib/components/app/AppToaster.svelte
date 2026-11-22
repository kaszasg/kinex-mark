<script lang="ts" module>
	/**
	 * Per-type toast classes: the Lucide icon takes the status colour (plan 8.7.0), the text stays
	 * popover-foreground. `cn-toast` is the generated Toaster's own toast class, kept because these
	 * options replace the generated ones.
	 */
	export const TOAST_CLASSES = {
		toast: 'cn-toast',
		success: '[&_[data-icon]]:text-success',
		info: '[&_[data-icon]]:text-info',
		warning: '[&_[data-icon]]:text-warning',
		error: '[&_[data-icon]]:text-destructive'
	} as const;
</script>

<script lang="ts">
	import { Toaster } from '$lib/components/ui/sonner/index.js';
	import { t } from '$lib/i18n/index.js';

	/**
	 * The toast region (plan 8.7.0): bottom right, 4 s. Its accessible names follow the active
	 * language; svelte-sonner's defaults are English ("Notifications", "Close toast").
	 */
</script>

<Toaster
	position="bottom-right"
	duration={4000}
	containerAriaLabel={t('common.notifications')}
	closeButtonAriaLabel={t('common.actions.close')}
	toastOptions={{ classes: TOAST_CLASSES }}
/>
