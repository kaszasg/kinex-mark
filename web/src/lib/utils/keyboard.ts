// Keyboard shortcut guards and key labels (plan 8.7.3).
import { t, type MessageKey } from '$lib/i18n/index.js';

/** Reports whether the event target accepts text: input, textarea, select or contenteditable. */
export function isEditableElement(target: EventTarget | null): boolean {
	if (!(target instanceof HTMLElement)) return false;
	if (target.isContentEditable) return true;
	const tag = target.tagName;
	if (tag === 'TEXTAREA' || tag === 'SELECT') return true;
	if (tag === 'INPUT') {
		const type = (target as HTMLInputElement).type;
		return !['button', 'checkbox', 'radio', 'submit', 'reset', 'range', 'color', 'file'].includes(
			type
		);
	}
	return target.closest('[contenteditable=""], [contenteditable="true"]') !== null;
}

/** Reports whether a modal dialog, alert dialog or sheet is open. */
export function isDialogOpen(root: ParentNode = document): boolean {
	return root.querySelector('[role="dialog"], [role="alertdialog"]') !== null;
}

/**
 * A single-key shortcut runs only when focus is not in an editable element, no dialog is
 * open and no Ctrl, Alt or Meta modifier is held (paste is handled separately).
 */
export function shouldHandleShortcut(event: KeyboardEvent): boolean {
	if (event.defaultPrevented || event.isComposing) return false;
	if (event.ctrlKey || event.altKey || event.metaKey) return false;
	if (isEditableElement(event.target)) return false;
	return !isDialogOpen();
}

/** Reports whether the platform uses the Command key as the primary modifier. */
export function isApplePlatform(): boolean {
	if (typeof navigator === 'undefined') return false;
	const platform =
		(navigator as Navigator & { userAgentData?: { platform?: string } }).userAgentData?.platform ??
		navigator.platform ??
		'';
	return /mac|iphone|ipad|ipod/i.test(platform);
}

/**
 * The label of the primary modifier key: the Command sign on Apple platforms, elsewhere the
 * Control key as the active language's keyboards print it (Ctrl, German Strg).
 */
export function modifierKeyLabel(): string {
	return isApplePlatform() ? '⌘' : t('common.keys.ctrl');
}

/** Named keys of the shortcut table, printed in the active language (German Entf, ...). */
const KEY_LABELS: Partial<Record<string, MessageKey>> = {
	Enter: 'common.keys.enter',
	Delete: 'common.keys.delete',
	Esc: 'common.keys.esc'
};

/** The label of one key of a shortcut: a named key in the active language, others as given. */
export function keyLabel(key: string): string {
	const message = KEY_LABELS[key];
	return message ? t(message) : key;
}
