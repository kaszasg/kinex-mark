// src/app.css must stay identical (comments aside) to the brand stylesheet in ai/GRAPHICS.md 13.2.
import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

// Vitest runs with web/ as the working directory (import.meta.url is not a file URL under jsdom).
const appCssPath = resolve('src/app.css');
const graphicsPath = resolve('../ai/GRAPHICS.md');
const HEADER =
	'/* KINEXIUM brand stylesheet. Generated from ai/GRAPHICS.md section 13.2 by scripts/sync-app-css.mjs; do not edit by hand. */';

/** Removes CSS comments and collapses every whitespace run to one space. */
function normalize(css: string): string {
	return css
		.replace(/\/\*[\s\S]*?\*\//g, ' ')
		.replace(/\s+/g, ' ')
		.trim();
}

/** Returns the first css code block after the 13.2 heading, located by its fences. */
function brandBlock(markdown: string): string {
	const lines = markdown.replace(/\r\n?/g, '\n').split('\n');
	const heading = lines.findIndex((line) => line.startsWith('### 13.2'));
	const open = lines.findIndex((line, i) => i > heading && line.trim() === '```css');
	const close = lines.findIndex((line, i) => i > open && line.trim() === '```');
	if (heading < 0 || open < 0 || close < 0) {
		throw new Error('css block of section 13.2 not found in GRAPHICS.md');
	}
	return lines.slice(open + 1, close).join('\n');
}

describe('src/app.css', () => {
	const appCss = readFileSync(appCssPath, 'utf8');

	it('starts with the generated-file header and holds no other comment', () => {
		expect(appCss.split('\n')[0]).toBe(HEADER);
		expect(appCss.match(/\/\*/g)).toHaveLength(1);
	});

	it('contains ASCII characters only', () => {
		expect(appCss).toMatch(/^[\x09\x0a\x0d\x20-\x7e]*$/);
	});

	describe.skipIf(!existsSync(graphicsPath))('compared with ai/GRAPHICS.md', () => {
		it('equals the section 13.2 block without comments', () => {
			const expected = normalize(brandBlock(readFileSync(graphicsPath, 'utf8')));
			expect(expected.length).toBeGreaterThan(1000);
			expect(normalize(appCss)).toBe(expected);
		});
	});
});
