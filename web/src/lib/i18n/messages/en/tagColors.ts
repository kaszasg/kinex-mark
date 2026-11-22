import type hu from '../hu/tagColors.js';
import type { Shape } from '../../types.js';

// Names of the seven tag colours in the order of plan 4.6 (F3).
export default {
	blue: 'Blue',
	ochre: 'Ochre',
	green: 'Green',
	violet: 'Violet',
	rose: 'Rose',
	neutral: 'Neutral',
	contrast: 'Contrast'
} satisfies Shape<typeof hu>;
