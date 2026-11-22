// Sample label of the display settings (plan 8.7.3 precedence on fixed texts).
import { describe, expect, test } from 'vitest';
import { previewLines, type DisplayFields } from './preview.js';

const sample = {
	title: 'Grafana',
	description: 'Éles műszerfal',
	url: 'grafana.example.net/d/abc'
};
const all: DisplayFields = {
	showIcon: true,
	showUrl: true,
	showTitle: true,
	showDescription: true
};

describe('previewLines', () => {
	test('every field on: description label, title and URL lines', () => {
		expect(previewLines(sample, all)).toEqual({
			icon: true,
			label: 'Éles műszerfal',
			labelSource: 'description',
			title: 'Grafana',
			url: 'grafana.example.net/d/abc'
		});
	});

	test('description off: the title becomes the label', () => {
		expect(previewLines(sample, { ...all, showDescription: false })).toEqual({
			icon: true,
			label: 'Grafana',
			labelSource: 'title',
			url: 'grafana.example.net/d/abc'
		});
	});

	test('description and title off: the URL is the label and no line repeats it', () => {
		expect(
			previewLines(sample, { ...all, showDescription: false, showTitle: false, showIcon: false })
		).toEqual({ icon: false, label: 'grafana.example.net/d/abc', labelSource: 'url' });
	});

	test('URL off hides the URL line; title off hides the title line', () => {
		expect(previewLines(sample, { ...all, showUrl: false, showTitle: false })).toEqual({
			icon: true,
			label: 'Éles műszerfal',
			labelSource: 'description'
		});
	});

	test('empty texts fall through and a title equal to the label is not repeated', () => {
		expect(previewLines({ ...sample, description: '  ' }, all).labelSource).toBe('title');
		expect(previewLines({ ...sample, description: 'grafana' }, all).title).toBeUndefined();
	});
});
