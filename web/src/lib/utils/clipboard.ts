// Clipboard writing with a fallback for insecure contexts (plan 8.7.3).
import { toast } from 'svelte-sonner';
import { t } from '$lib/i18n/index.js';

function fallbackCopy(text: string): boolean {
	const area = document.createElement('textarea');
	area.value = text;
	area.setAttribute('readonly', '');
	area.style.position = 'fixed';
	area.style.top = '0';
	area.style.left = '0';
	area.style.opacity = '0';
	document.body.appendChild(area);
	const active = document.activeElement as HTMLElement | null;
	area.select();
	let ok = false;
	try {
		ok = document.execCommand('copy');
	} catch {
		ok = false;
	} finally {
		area.remove();
		active?.focus?.();
	}
	return ok;
}

/**
 * Copies text to the clipboard: navigator.clipboard when available, otherwise a hidden
 * textarea with execCommand. On failure an error toast asks the user to copy by hand.
 * Resolves true on success.
 */
export async function copyText(text: string): Promise<boolean> {
	let ok = false;
	if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
		try {
			await navigator.clipboard.writeText(text);
			ok = true;
		} catch {
			ok = fallbackCopy(text);
		}
	} else {
		ok = fallbackCopy(text);
	}
	if (!ok) toast.error(t('common.copyFailed'));
	return ok;
}
