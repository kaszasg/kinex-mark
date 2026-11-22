// src/kinex-mark.css print rules (plan 8.1.6 item 4): the recovery-code sheet prints dark ink on
// a light page in both themes, whichever surface token the printed elements use.
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

// Vitest runs with web/ as the working directory.
const css = readFileSync(resolve('src/kinex-mark.css'), 'utf8').replace(/\/\*[\s\S]*?\*\//g, '');
const brand = readFileSync(resolve('src/app.css'), 'utf8').replace(/\/\*[\s\S]*?\*\//g, '');

/** The declarations of the first rule whose selector list starts with the given text. */
function declarations(source: string, selectorStart: string): Map<string, string> {
	const start = source.indexOf(selectorStart);
	if (start < 0) throw new Error(`rule ${selectorStart} not found`);
	const open = source.indexOf('{', start);
	const close = source.indexOf('}', open);
	const out = new Map<string, string>();
	for (const part of source.slice(open + 1, close).split(';')) {
		const colon = part.indexOf(':');
		if (colon < 0) continue;
		out.set(part.slice(0, colon).trim(), part.slice(colon + 1).trim());
	}
	return out;
}

/** The neutral colour tokens of the dark theme: surfaces, their text colours, borders, fields. */
function neutralTokens(): string[] {
	const dark = declarations(brand, ':root, .dark');
	const accentish =
		/^--(primary|destructive|success|warning|info|ring|selection|glow|sidebar|shadow)/;
	return [...dark.keys()].filter(
		(name) =>
			/^--(background|foreground|card|popover|secondary|muted|accent|border|input)/.test(name) &&
			!accentish.test(name)
	);
}

describe('src/kinex-mark.css print rules', () => {
	const print = css.slice(css.indexOf('@media print'));
	const paper = declarations(print, ':root');

	it('sees the neutral tokens of the brand stylesheet', () => {
		expect(neutralTokens()).toEqual(
			expect.arrayContaining(['--popover', '--popover-foreground', '--muted', '--foreground'])
		);
	});

	it('resets every neutral surface to the light mist and every text to the dark ink (E2)', () => {
		for (const token of neutralTokens()) {
			const value = paper.get(token);
			const expected =
				/foreground$/.test(token) || token === '--border' || token === '--input'
					? 'var(--ink-950)'
					: 'var(--mist-50)';
			expect(value, token).toBe(expected);
		}
	});

	it('prints in the light colour scheme', () => {
		expect(paper.get('color-scheme')).toBe('light');
	});
});
