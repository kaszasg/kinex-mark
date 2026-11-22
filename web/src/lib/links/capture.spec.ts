import { describe, expect, it } from 'vitest';
import {
	anchorTitle,
	containsUrl,
	extractCaptureItems,
	INTERNAL_DRAG_TYPE,
	isCaptureDrag,
	isEditableTarget,
	limitCaptureItems,
	MAX_CAPTURE_ITEMS,
	MAX_TEXT_LENGTH,
	MAX_TITLE_LENGTH
} from './capture.js';

function transfer(data: Record<string, string>) {
	return {
		types: Object.keys(data),
		getData: (type: string) => data[type] ?? ''
	};
}

describe('isCaptureDrag', () => {
	it('accepts URI lists and plain text', () => {
		expect(isCaptureDrag(['text/uri-list'])).toBe(true);
		expect(isCaptureDrag(['text/plain', 'text/html'])).toBe(true);
	});

	it('rejects files, own links, other types and missing types', () => {
		expect(isCaptureDrag(['Files', 'text/plain'])).toBe(false);
		expect(isCaptureDrag([INTERNAL_DRAG_TYPE, 'text/uri-list'])).toBe(false);
		expect(isCaptureDrag(['text/html'])).toBe(false);
		expect(isCaptureDrag(null)).toBe(false);
		expect(isCaptureDrag(undefined)).toBe(false);
	});
});

describe('anchorTitle', () => {
	it('returns the trimmed text of the first anchor', () => {
		expect(anchorTitle('<p><a href="x">  Grafana \n board </a><a>Other</a></p>')).toBe(
			'Grafana board'
		);
	});

	it('returns undefined without an anchor text', () => {
		expect(anchorTitle('')).toBeUndefined();
		expect(anchorTitle('<p>no anchor</p>')).toBeUndefined();
		expect(anchorTitle('<a href="x">   </a>')).toBeUndefined();
	});

	it('cuts the title at 500 code points', () => {
		const long = '😀'.repeat(MAX_TITLE_LENGTH + 10);
		expect([...(anchorTitle(`<a>${long}</a>`) ?? '')].length).toBe(MAX_TITLE_LENGTH);
	});
});

describe('extractCaptureItems', () => {
	it('splits a URI list and skips comments and blank lines', () => {
		const items = extractCaptureItems(
			transfer({ 'text/uri-list': '# comment\r\nhttps://a.io\n\nhttps://b.io\r' })
		);
		expect(items).toEqual([{ text: 'https://a.io' }, { text: 'https://b.io' }]);
	});

	it('falls back to plain text and takes the title from the HTML anchor of a single item', () => {
		const items = extractCaptureItems(
			transfer({
				'text/plain': '  see https://a.io  ',
				'text/html': '<a href="https://a.io">Docs</a>'
			})
		);
		expect(items).toEqual([{ text: 'see https://a.io', title: 'Docs' }]);
	});

	it('ignores the HTML title when there are several items', () => {
		const items = extractCaptureItems(
			transfer({
				'text/uri-list': 'https://a.io\nhttps://b.io',
				'text/html': '<a>Docs</a>'
			})
		);
		expect(items.every((item) => item.title === undefined)).toBe(true);
	});

	it('uses plain text when the URI list has only comments', () => {
		const items = extractCaptureItems(
			transfer({ 'text/uri-list': '# only', 'text/plain': 'https://c.io' })
		);
		expect(items).toEqual([{ text: 'https://c.io' }]);
	});

	it('returns nothing for empty text and for rejected drags', () => {
		expect(extractCaptureItems(transfer({ 'text/plain': '   ' }))).toEqual([]);
		expect(extractCaptureItems(transfer({ Files: '', 'text/plain': 'https://a.io' }))).toEqual([]);
	});

	it('cuts the text at the server limit', () => {
		const items = extractCaptureItems(transfer({ 'text/plain': 'x'.repeat(MAX_TEXT_LENGTH + 5) }));
		expect(items[0].text).toHaveLength(MAX_TEXT_LENGTH);
	});
});

describe('limitCaptureItems', () => {
	it('keeps the first 20 and counts the rest', () => {
		const items = Array.from({ length: 23 }, (_, index) => ({ text: `https://a.io/${index}` }));
		const result = limitCaptureItems(items);
		expect(result.items).toHaveLength(MAX_CAPTURE_ITEMS);
		expect(result.skipped).toBe(3);
		expect(limitCaptureItems(items.slice(0, 2)).skipped).toBe(0);
	});
});

describe('containsUrl', () => {
	it.each([
		['https://a.io', true],
		['look at http://a.io/x please', true],
		['example.com/path', true],
		['bücher.example', true],
		['localhost:8080', false],
		['just words', false],
		['# https://a.io', false],
		['', false]
	])('%s -> %s', (text, expected) => {
		expect(containsUrl(text)).toBe(expected);
	});
});

describe('isEditableTarget', () => {
	it('detects text fields and ignores buttons', () => {
		const input = document.createElement('input');
		const button = document.createElement('button');
		expect(isEditableTarget(input)).toBe(true);
		expect(isEditableTarget(button)).toBe(false);
		expect(isEditableTarget(null)).toBe(false);
	});
});
