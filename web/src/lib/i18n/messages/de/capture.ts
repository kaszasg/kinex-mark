import type hu from '../hu/capture.js';
import type { Shape } from '../../types.js';

export default {
	overlay: {
		title: 'Loslassen, um den Link zu speichern',
		text: 'Eine URL, ein Browser-Link oder Text mit einer URL. Titel und Symbol kommen im Hintergrund.'
	},
	toast: {
		saved: 'Link gespeichert',
		exists: 'Diesen Link hast du schon',
		multiple: {
			one: '{count} Link gespeichert, {existing} schon vorhanden',
			other: '{count} Links gespeichert, {existing} schon vorhanden'
		},
		hidden: 'Link gespeichert. Die aktuellen Filter blenden ihn aus.',
		tooMany: 'Du kannst höchstens {max} Links auf einmal speichern, der Rest wurde übersprungen.',
		invalidUrl: 'Die gefundene URL kann nicht gespeichert werden.',
		openAction: 'Öffnen',
		editAction: 'Bearbeiten',
		showAction: 'Zeigen'
	}
} satisfies Shape<typeof hu>;
