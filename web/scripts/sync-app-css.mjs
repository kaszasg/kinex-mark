// Regenerates src/app.css from the brand stylesheet in ai/GRAPHICS.md (section 13.2).
// The source block carries Hungarian comments; source code must be English only,
// so every CSS comment is removed and a single English header is prepended.
// Usage (from web/): node scripts/sync-app-css.mjs
import { readFileSync, writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

const HEADING = '### 13.2';
const HEADER =
	'/* KINEXIUM brand stylesheet. Generated from ai/GRAPHICS.md section 13.2 by scripts/sync-app-css.mjs; do not edit by hand. */';

const sourcePath = fileURLToPath(new URL('../../ai/GRAPHICS.md', import.meta.url));
const targetPath = fileURLToPath(new URL('../src/app.css', import.meta.url));

/**
 * Returns the body of the first ```css fenced block that follows the 13.2 heading.
 * The block is located by its fences, never by line numbers.
 * @param {string} markdown
 * @returns {string}
 */
export function extractCssBlock(markdown) {
	const lines = markdown.replace(/\r\n?/g, '\n').split('\n');
	const headingIndex = lines.findIndex((line) => line.startsWith(HEADING));
	if (headingIndex < 0) {
		throw new Error(`heading "${HEADING}" not found in GRAPHICS.md`);
	}
	const openIndex = lines.findIndex((line, i) => i > headingIndex && line.trim() === '```css');
	if (openIndex < 0) {
		throw new Error(`no css code block after heading "${HEADING}"`);
	}
	const closeIndex = lines.findIndex((line, i) => i > openIndex && line.trim() === '```');
	if (closeIndex < 0) {
		throw new Error('unterminated css code block in GRAPHICS.md');
	}
	return lines.slice(openIndex + 1, closeIndex).join('\n');
}

/**
 * Removes every CSS comment. Lines that held only a comment disappear,
 * trailing comments are cut together with the whitespace before them.
 * @param {string} css
 * @returns {string}
 */
export function stripComments(css) {
	return css
		.replace(/^[ \t]*\/\*[\s\S]*?\*\/[ \t]*\n/gm, '')
		.replace(/[ \t]*\/\*[\s\S]*?\*\//g, '')
		.split('\n')
		.map((line) => line.trimEnd())
		.join('\n')
		.trim();
}

/**
 * Builds the final stylesheet text from GRAPHICS.md content.
 * @param {string} markdown
 * @returns {string}
 */
export function buildAppCss(markdown) {
	return `${HEADER}\n${stripComments(extractCssBlock(markdown))}\n`;
}

function main() {
	const css = buildAppCss(readFileSync(sourcePath, 'utf8'));
	writeFileSync(targetPath, css, 'utf8');
	console.log(`wrote ${targetPath} (${css.split('\n').length - 1} lines)`);
}

main();
