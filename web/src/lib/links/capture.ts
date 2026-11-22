// Quick capture from drag and drop or paste (plan 8.7.3, 5.9.3).
import { isEditableElement } from '$lib/utils/keyboard.js';

/** Set on drags that start from a link of this app, so dropping it back does not capture it. */
export const INTERNAL_DRAG_TYPE = 'application/x-kinex-mark';
/** At most this many links are saved from one drop. */
export const MAX_CAPTURE_ITEMS = 20;
/** The server's limit of the anchor title (plan 5.9.3). */
export const MAX_TITLE_LENGTH = 500;
/** The server's limit of the captured text (plan 5.9.3). */
export const MAX_TEXT_LENGTH = 8192;

export type CaptureItem = { text: string; title?: string };

type TransferLike = Pick<DataTransfer, 'types' | 'getData'>;

function hasType(types: readonly string[] | DOMStringList, type: string): boolean {
	return Array.from(types as ArrayLike<string>).includes(type);
}

/**
 * A drag the capture accepts: it carries a URI list or plain text, and neither files nor a link
 * dragged from this app.
 */
export function isCaptureDrag(
	types: readonly string[] | DOMStringList | null | undefined
): boolean {
	if (!types) return false;
	if (hasType(types, 'Files') || hasType(types, INTERNAL_DRAG_TYPE)) return false;
	return hasType(types, 'text/uri-list') || hasType(types, 'text/plain');
}

/** Anchor text of the first <a> in text/html, trimmed, max 500 characters. */
export function anchorTitle(html: string): string | undefined {
	if (!html || typeof DOMParser === 'undefined') return undefined;
	const doc = new DOMParser().parseFromString(html, 'text/html');
	const text = doc.querySelector('a')?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
	if (!text) return undefined;
	return [...text].slice(0, MAX_TITLE_LENGTH).join('');
}

/** URL candidates from a DataTransfer: text/uri-list lines (no "#" comments), else text/plain. */
export function extractCaptureItems(dt: TransferLike): CaptureItem[] {
	if (!isCaptureDrag(dt.types)) return [];
	let texts: string[] = [];
	if (hasType(dt.types, 'text/uri-list')) {
		texts = dt
			.getData('text/uri-list')
			.split(/\r\n|\r|\n/)
			.map((line) => line.trim())
			.filter((line) => line !== '' && !line.startsWith('#'));
	}
	if (texts.length === 0 && hasType(dt.types, 'text/plain')) {
		const text = dt.getData('text/plain').trim();
		if (text) texts = [text];
	}
	const items: CaptureItem[] = texts.map((text) => ({ text: text.slice(0, MAX_TEXT_LENGTH) }));
	if (items.length === 1 && hasType(dt.types, 'text/html')) {
		const title = anchorTitle(dt.getData('text/html'));
		if (title) items[0].title = title;
	}
	return items;
}

/** The first 20 items to save and the number of the skipped rest. */
export function limitCaptureItems(items: CaptureItem[]): { items: CaptureItem[]; skipped: number } {
	return {
		items: items.slice(0, MAX_CAPTURE_ITEMS),
		skipped: Math.max(0, items.length - MAX_CAPTURE_ITEMS)
	};
}

const SCHEME_URL = /https?:\/\/\S/i;
const DOMAIN_TOKEN = /^[\p{L}\p{N}.-]+\.[\p{L}]{2,63}(:\d+)?(\/\S*)?$/u;

/**
 * Mirrors the server's URL extraction (fetch.ExtractURL) closely enough to skip a request that
 * would surely fail: an http(s) URL anywhere, or a domain-shaped first token.
 */
export function containsUrl(text: string): boolean {
	const lines = text.split(/\r\n|\r|\n/).filter((line) => !line.trim().startsWith('#'));
	const content = lines.join('\n');
	if (SCHEME_URL.test(content)) return true;
	const first = content.trim().split(/\s+/)[0] ?? '';
	return DOMAIN_TOKEN.test(first);
}

/** input, textarea, select or contenteditable: paste there stays a normal paste. */
export function isEditableTarget(target: EventTarget | null): boolean {
	return isEditableElement(target);
}
