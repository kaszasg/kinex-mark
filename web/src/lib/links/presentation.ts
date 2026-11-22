// Display label of a link (plan 8.7.3): description, otherwise title, otherwise the URL,
// narrowed by the user's display switches. Pure functions, no DOM.
import type { LinkItem } from '$lib/api/types.js';

export type DisplayFields = {
	showIcon: boolean;
	showUrl: boolean;
	showTitle: boolean;
	showDescription: boolean;
};

export type Presentation = {
	label: string;
	labelSource: 'description' | 'title' | 'url';
	secondary: { title?: string; description?: string; url?: string };
};

/** Longer URLs collapse to the host in the label. */
export const LABEL_URL_MAX = 40;
/** Longer URLs collapse to the host in the URL line. */
export const LINE_URL_MAX = 64;

/** Every field on: the full precedence used by Markdown copy and the delete confirmation. */
export const ALL_FIELDS: DisplayFields = {
	showIcon: true,
	showUrl: true,
	showTitle: true,
	showDescription: true
};

const URL_PARTS = /^[a-z][a-z0-9+.-]*:\/\/([^/?#]*)(.*)$/i;

/** Length in Unicode code points. */
export function codePointLength(value: string): number {
	return [...value].length;
}

function decode(value: string): string {
	try {
		return decodeURIComponent(value);
	} catch {
		return value;
	}
}

function displayPath(rest: string): string {
	try {
		return decodeURI(rest);
	} catch {
		return rest;
	}
}

/** A host for display: lower case, percent-decoded, without the "www." prefix. */
export function prettyHost(host: string): string {
	const lower = decode(host.trim()).toLowerCase();
	return lower.startsWith('www.') ? lower.slice(4) : lower;
}

/** "https://Example.com/a/b/" -> "example.com/a/b": no scheme, no "www.", no trailing slash. */
export function prettyUrl(url: string): string {
	const trimmed = url.trim();
	const match = URL_PARTS.exec(trimmed);
	if (!match) return trimmed;
	const host = prettyHost(match[1]);
	let rest = displayPath(match[2]);
	if (rest.endsWith('/')) rest = rest.slice(0, -1);
	return host + rest;
}

/** A run of percent-encoded bytes. */
const PERCENT_RUN = /(?:%[0-9a-f]{2})+/gi;
/**
 * Characters that stay percent-encoded in a readable URL: controls, format characters (the
 * bidi controls among them), separators and white space, private-use and unassigned code points.
 * Decoded they would be invisible or could reorder the text on screen.
 */
const HIDDEN_RUNE = /[\p{Cc}\p{Cf}\p{Z}\p{Co}\p{Cn}\s]/u;
const utf8 = new TextDecoder('utf-8', { fatal: true });

/** Length of the UTF-8 sequence that a lead byte starts; 0 for a byte that cannot lead. */
function utf8Length(lead: number): number {
	if (lead >= 0xc2 && lead <= 0xdf) return 2;
	if (lead >= 0xe0 && lead <= 0xef) return 3;
	if (lead >= 0xf0 && lead <= 0xf4) return 4;
	return 0;
}

/** Decodes the non-ASCII UTF-8 characters of one percent-encoded run; ASCII escapes stay. */
function decodeRun(run: string): string {
	const escapes = run.match(/%[0-9a-f]{2}/gi) ?? [];
	const bytes = escapes.map((escape) => Number.parseInt(escape.slice(1), 16));
	let out = '';
	let i = 0;
	while (i < bytes.length) {
		const size = utf8Length(bytes[i]);
		if (size > 0 && i + size <= bytes.length) {
			try {
				const char = utf8.decode(new Uint8Array(bytes.slice(i, i + size)));
				if (!HIDDEN_RUNE.test(char)) {
					out += char;
					i += size;
					continue;
				}
			} catch {
				// Not valid UTF-8: the escapes stay as they are.
			}
		}
		out += escapes[i];
		i += 1;
	}
	return out;
}

/**
 * The full URL for reading and editing (details sheet, edit dialog): scheme, host, path, query
 * and fragment as stored, but with percent-encoded non-ASCII letters shown as letters, as the
 * list shows them ("https://b%C3%BCcher.example" reads as the IDN host with the u-umlaut). Escapes
 * of ASCII characters (%2F, %25, %20, ...) and of invisible characters stay, so the server
 * normalises the text back to the same URL.
 */
export function readableUrl(url: string): string {
	return url.replace(PERCENT_RUN, decodeRun);
}

/** prettyUrl when it fits in max characters, otherwise the host only. */
export function compactUrl(url: string, host: string, max: number): string {
	const pretty = prettyUrl(url);
	if (codePointLength(pretty) <= max) return pretty;
	const shortHost = prettyHost(host);
	return shortHost || pretty;
}

function firstLine(value: string): string {
	return (
		value
			.split(/\r\n|\r|\n/)
			.map((line) => line.trim())
			.find((line) => line !== '') ?? ''
	);
}

export function linkPresentation(
	link: Pick<LinkItem, 'url' | 'host' | 'title' | 'description'>,
	fields: DisplayFields
): Presentation {
	const title = link.title.trim();
	const description = link.description.trim();
	const descriptionLine = firstLine(description);

	let label: string;
	let labelSource: Presentation['labelSource'];
	if (fields.showDescription && descriptionLine) {
		label = descriptionLine;
		labelSource = 'description';
	} else if (fields.showTitle && title) {
		label = title;
		labelSource = 'title';
	} else {
		label = compactUrl(link.url, link.host, LABEL_URL_MAX);
		labelSource = 'url';
	}

	const secondary: Presentation['secondary'] = {};
	if (
		fields.showTitle &&
		title &&
		labelSource !== 'title' &&
		title.toLowerCase() !== label.toLowerCase()
	) {
		secondary.title = title;
	}
	if (fields.showDescription && description && labelSource !== 'description') {
		secondary.description = description;
	}
	if (fields.showUrl && labelSource !== 'url') {
		secondary.url = compactUrl(link.url, link.host, LINE_URL_MAX);
	}
	return { label, labelSource, secondary };
}

/** The label with every field on, independent of the user's switches. */
export function fullLabel(link: Pick<LinkItem, 'url' | 'host' | 'title' | 'description'>): string {
	return linkPresentation(link, ALL_FIELDS).label;
}
