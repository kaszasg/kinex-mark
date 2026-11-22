// Sample link label of the display settings: shows how the four field switches shape a row.
// It follows the precedence of plan 8.7.3 (description, then title, then URL) on fixed sample
// texts; the links list itself uses linkPresentation() of $lib/links/presentation.ts (F2).
import type { Preferences } from '$lib/api/types.js';

export type DisplayFields = Pick<
	Preferences,
	'showIcon' | 'showUrl' | 'showTitle' | 'showDescription'
>;

export type PreviewSample = { title: string; description: string; url: string };

export type PreviewLines = {
	icon: boolean;
	label: string;
	labelSource: 'description' | 'title' | 'url';
	/** Secondary title line: only when the description is the label. */
	title?: string;
	/** Secondary URL line: only when the label is not the URL itself. */
	url?: string;
};

/** The label and the secondary lines a row shows for the sample under the given switches. */
export function previewLines(sample: PreviewSample, fields: DisplayFields): PreviewLines {
	const description = sample.description.trim();
	const title = sample.title.trim();
	const lines: PreviewLines = { icon: fields.showIcon, label: sample.url, labelSource: 'url' };
	if (fields.showDescription && description) {
		lines.label = description;
		lines.labelSource = 'description';
		if (fields.showTitle && title && title.toLowerCase() !== description.toLowerCase()) {
			lines.title = title;
		}
	} else if (fields.showTitle && title) {
		lines.label = title;
		lines.labelSource = 'title';
	}
	if (fields.showUrl && lines.labelSource !== 'url') lines.url = sample.url;
	return lines;
}
