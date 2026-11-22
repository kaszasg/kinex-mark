import type hu from '../hu/roles.js';
import type { Shape } from '../../types.js';

export default {
	admin: 'Administrator',
	developer: 'Developer',
	customer: 'Customer',
	plural: {
		admin: 'Administrators',
		developer: 'Developers',
		customer: 'Customers'
	}
} satisfies Shape<typeof hu>;
