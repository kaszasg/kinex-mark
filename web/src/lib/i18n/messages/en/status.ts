import type hu from '../hu/status.js';
import type { Shape } from '../../types.js';

export default {
	online: 'Online',
	warning: 'Warning',
	critical: 'Critical',
	offline: 'Offline',
	info: 'Syncing',
	active: 'Active',
	inactive: 'Inactive',
	locked: 'Locked',
	enabled: 'On',
	disabled: 'Off'
} satisfies Shape<typeof hu>;
