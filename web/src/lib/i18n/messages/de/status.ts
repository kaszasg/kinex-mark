import type hu from '../hu/status.js';
import type { Shape } from '../../types.js';

export default {
	online: 'Online',
	warning: 'Warnung',
	critical: 'Kritisch',
	offline: 'Offline',
	info: 'Synchronisiert',
	active: 'Aktiv',
	inactive: 'Inaktiv',
	locked: 'Gesperrt',
	enabled: 'Aktiviert',
	disabled: 'Deaktiviert'
} satisfies Shape<typeof hu>;
