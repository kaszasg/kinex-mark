import type hu from '../hu/roles.js';
import type { Shape } from '../../types.js';

export default {
	admin: 'Administrator',
	developer: 'Entwickler',
	customer: 'Kunde',
	plural: {
		admin: 'Administratoren',
		developer: 'Entwickler',
		customer: 'Kunden'
	}
} satisfies Shape<typeof hu>;
