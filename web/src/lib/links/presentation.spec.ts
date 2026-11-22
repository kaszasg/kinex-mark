import { describe, expect, it } from 'vitest';
import {
	ALL_FIELDS,
	codePointLength,
	compactUrl,
	fullLabel,
	LABEL_URL_MAX,
	linkPresentation,
	prettyHost,
	prettyUrl,
	readableUrl,
	type DisplayFields
} from './presentation.js';

type Row = { description: string; title: string; url: string; host: string };

function link(row: Partial<Row> & { url: string }): Row {
	return {
		description: '',
		title: '',
		host: new URL(row.url).host,
		...row
	};
}

const off = (patch: Partial<DisplayFields>): DisplayFields => ({ ...ALL_FIELDS, ...patch });

describe('prettyUrl', () => {
	it.each([
		['https://Example.com/a/b/', 'example.com/a/b'],
		['https://www.example.com/', 'example.com'],
		['http://example.com', 'example.com'],
		['https://example.com/path?q=1#top', 'example.com/path?q=1#top'],
		['https://example.com/caf%C3%A9', 'example.com/café'],
		['https://xn--bcher-kva.example/', 'xn--bcher-kva.example'],
		['https://bücher.example/regal/', 'bücher.example/regal'],
		['https://example.com/%E0%A4%A', 'example.com/%E0%A4%A'],
		['not a url', 'not a url']
	])('formats %s as %s', (input, expected) => {
		expect(prettyUrl(input)).toBe(expected);
	});
});

describe('readableUrl', () => {
	it.each([
		// The IDN host and path of the E2 report: letters decoded, the query kept as stored.
		[
			'https://b%C3%BCcher.example.invalid/stra%C3%9Fe/%C3%B6ffnungszeiten?ő=ű',
			'https://bücher.example.invalid/straße/öffnungszeiten?ő=ű'
		],
		['https://example.com/caf%C3%A9#t%C3%B6p', 'https://example.com/café#töp'],
		// ASCII escapes stay, so the server reads the text back as the same URL.
		['https://example.com/a%2Fb%20c%25d?q=%26', 'https://example.com/a%2Fb%20c%25d?q=%26'],
		// Invisible characters stay encoded: no-break space, RLO, zero-width space, a C1 control.
		['https://example.com/x%C2%A0y', 'https://example.com/x%C2%A0y'],
		['https://example.com/%E2%80%AEgpj', 'https://example.com/%E2%80%AEgpj'],
		['https://example.com/a%E2%80%8Bb', 'https://example.com/a%E2%80%8Bb'],
		['https://example.com/%C2%85', 'https://example.com/%C2%85'],
		// Broken UTF-8 stays as it is; a valid character after it is still decoded.
		['https://example.com/%E0%A4%A', 'https://example.com/%E0%A4%A'],
		['https://example.com/%C3%28%C3%A9', 'https://example.com/%C3%28é'],
		['https://example.com/%F0%9F%93%9A', 'https://example.com/📚'],
		['https://example.com/plain', 'https://example.com/plain']
	])('shows %s as %s', (input, expected) => {
		expect(readableUrl(input)).toBe(expected);
	});

	it('matches the host and path of the list line', () => {
		const url = 'https://b%C3%BCcher.example.invalid/stra%C3%9Fe/%C3%B6ffnungszeiten?ő=ű';
		expect(readableUrl(url)).toBe(`https://${prettyUrl(url)}`);
	});
});

describe('prettyHost', () => {
	it('lower-cases, decodes and drops www.', () => {
		expect(prettyHost(' WWW.Example.COM ')).toBe('example.com');
		expect(prettyHost('b%C3%BCcher.example')).toBe('bücher.example');
		expect(prettyHost('%E0%A4%A')).toBe('%e0%a4%a');
	});
});

describe('compactUrl', () => {
	it('keeps a URL of exactly 40 code points and collapses one of 41 to the host', () => {
		const path40 = 'a.io/' + 'x'.repeat(35);
		const path41 = 'a.io/' + 'x'.repeat(36);
		expect(codePointLength(path40)).toBe(LABEL_URL_MAX);
		expect(compactUrl(`https://${path40}`, 'a.io', LABEL_URL_MAX)).toBe(path40);
		expect(compactUrl(`https://${path41}`, 'a.io', LABEL_URL_MAX)).toBe('a.io');
	});

	it('counts code points, not UTF-16 units', () => {
		const path = 'a.io/' + '😀'.repeat(35);
		expect(path.length).toBeGreaterThan(40);
		expect(compactUrl(`https://${path}`, 'a.io', LABEL_URL_MAX)).toBe(path);
	});

	it('falls back to the pretty URL when the host is empty', () => {
		const long = 'https://a.io/' + 'y'.repeat(60);
		expect(compactUrl(long, '', 10)).toBe(prettyUrl(long));
	});
});

describe('linkPresentation: plan example table', () => {
	it('uses the description with title and URL lines', () => {
		const result = linkPresentation(
			link({
				description: 'Éles műszerfal',
				title: 'Grafana',
				url: 'https://grafana.example.net/d/abc'
			}),
			ALL_FIELDS
		);
		expect(result).toEqual({
			label: 'Éles műszerfal',
			labelSource: 'description',
			secondary: { title: 'Grafana', url: 'grafana.example.net/d/abc' }
		});
	});

	it('uses the title when there is no description', () => {
		const result = linkPresentation(
			link({ title: 'SvelteKit docs', url: 'https://svelte.dev/docs/kit' }),
			ALL_FIELDS
		);
		expect(result).toEqual({
			label: 'SvelteKit docs',
			labelSource: 'title',
			secondary: { url: 'svelte.dev/docs/kit' }
		});
	});

	it('uses the pretty URL without secondary lines', () => {
		const result = linkPresentation(link({ url: 'https://example.com/' }), ALL_FIELDS);
		expect(result).toEqual({ label: 'example.com', labelSource: 'url', secondary: {} });
	});

	it('collapses a long URL to the host', () => {
		const result = linkPresentation(
			link({ url: 'https://docs.example.com/very/long/path/to/a/page?id=12345&x=1' }),
			ALL_FIELDS
		);
		expect(result).toEqual({ label: 'docs.example.com', labelSource: 'url', secondary: {} });
	});

	it('skips the description when it is switched off', () => {
		const result = linkPresentation(
			link({ description: 'Leírás', title: 'Cím', url: 'https://a.io/x' }),
			off({ showDescription: false })
		);
		expect(result).toEqual({ label: 'Cím', labelSource: 'title', secondary: { url: 'a.io/x' } });
	});

	it('falls back to the URL when description and title are off', () => {
		const result = linkPresentation(
			link({ description: 'Leírás', title: 'Cím', url: 'https://a.io/x' }),
			off({ showDescription: false, showTitle: false })
		);
		expect(result).toEqual({ label: 'a.io/x', labelSource: 'url', secondary: {} });
	});
});

describe('linkPresentation: rules', () => {
	it('trims and takes the first non-empty line of a multi-line description', () => {
		const result = linkPresentation(
			link({ description: '\n  First line  \nSecond line', title: ' T ', url: 'https://a.io' }),
			ALL_FIELDS
		);
		expect(result.label).toBe('First line');
		expect(result.secondary.title).toBe('T');
		expect(result.secondary.description).toBeUndefined();
	});

	it('hides a title equal to the label, ignoring case', () => {
		const result = linkPresentation(
			link({ description: 'grafana', title: 'Grafana', url: 'https://a.io' }),
			ALL_FIELDS
		);
		expect(result.secondary.title).toBeUndefined();
	});

	it('never repeats a switched-off description as a secondary line', () => {
		const titled = linkPresentation(
			link({ description: 'Line one\nLine two', title: 'Title', url: 'https://a.io' }),
			off({ showDescription: false })
		);
		expect(titled.labelSource).toBe('title');
		expect(titled.secondary.description).toBeUndefined();
	});

	it('uses LINE_URL_MAX for the URL line', () => {
		const url = 'https://a.io/' + 'z'.repeat(50);
		const result = linkPresentation(link({ title: 'T', url }), ALL_FIELDS);
		expect(result.secondary.url).toBe('a.io/' + 'z'.repeat(50));
		const longer = 'https://a.io/' + 'z'.repeat(70);
		expect(linkPresentation(link({ title: 'T', url: longer }), ALL_FIELDS).secondary.url).toBe(
			'a.io'
		);
	});

	it('handles an IDN host', () => {
		const result = linkPresentation(
			link({ url: 'https://bücher.example/', host: 'bücher.example' }),
			ALL_FIELDS
		);
		expect(result.label).toBe('bücher.example');
	});

	it('fullLabel ignores the switches', () => {
		expect(fullLabel(link({ description: 'D', title: 'T', url: 'https://a.io' }))).toBe('D');
	});
});

describe('linkPresentation: all 16 switch combinations', () => {
	const sample = link({ description: 'Desc', title: 'Title', url: 'https://a.io/x' });
	const combos: DisplayFields[] = [];
	for (let mask = 0; mask < 16; mask++) {
		combos.push({
			showIcon: Boolean(mask & 1),
			showUrl: Boolean(mask & 2),
			showTitle: Boolean(mask & 4),
			showDescription: Boolean(mask & 8)
		});
	}

	it.each(combos)('follows the rules for %o', (fields) => {
		const result = linkPresentation(sample, fields);
		const expectedLabel = fields.showDescription ? 'Desc' : fields.showTitle ? 'Title' : 'a.io/x';
		expect(result.label).toBe(expectedLabel);
		expect(result.label).not.toBe('');
		const expectedSecondary: Record<string, string> = {};
		if (fields.showTitle && result.labelSource !== 'title') expectedSecondary.title = 'Title';
		if (fields.showUrl && result.labelSource !== 'url') expectedSecondary.url = 'a.io/x';
		expect(result.secondary).toEqual(expectedSecondary);
	});
});
