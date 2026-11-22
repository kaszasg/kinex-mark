import type hu from '../hu/tagColors.js';
import type { Shape } from '../../types.js';

// Names of the seven tag colours in the order of plan 4.6 (F3).
export default {
	blue: 'Blau',
	ochre: 'Ocker',
	green: 'Grün',
	violet: 'Violett',
	rose: 'Rosa',
	neutral: 'Neutral',
	contrast: 'Kontrast'
} satisfies Shape<typeof hu>;
