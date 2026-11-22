import { describe, expect, it } from 'vitest';
import { toMarkdownLink } from './markdown.js';

describe('toMarkdownLink', () => {
	it('builds the plan example', () => {
		expect(toMarkdownLink('Éles műszerfal', 'https://grafana.example.net/d/abc')).toBe(
			'[Éles műszerfal](https://grafana.example.net/d/abc)'
		);
	});

	it('escapes backslashes and brackets in the label', () => {
		expect(toMarkdownLink('a\\b [c]', 'https://a.io')).toBe('[a\\\\b \\[c\\]](https://a.io)');
	});

	it('turns line breaks into spaces', () => {
		expect(toMarkdownLink('one\r\ntwo\nthree\rfour', 'https://a.io')).toBe(
			'[one two three four](https://a.io)'
		);
	});

	it('encodes spaces and parentheses in the URL', () => {
		expect(toMarkdownLink('Wiki', 'https://a.io/Foo (bar) baz')).toBe(
			'[Wiki](https://a.io/Foo%20%28bar%29%20baz)'
		);
	});
});
