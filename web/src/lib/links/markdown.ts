// Markdown link text for "Copy as Markdown" (plan 8.7.3).

/** Escapes the label (backslash and brackets, line breaks to spaces) and the URL (space, parentheses). */
export function toMarkdownLink(label: string, url: string): string {
	const text = label.replace(/\r\n|\r|\n/g, ' ').replace(/[\\[\]]/g, (ch) => `\\${ch}`);
	const target = url.replace(/ /g, '%20').replace(/\(/g, '%28').replace(/\)/g, '%29');
	return `[${text}](${target})`;
}
