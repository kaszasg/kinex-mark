// The only tag colour mapping: tag colour -> swatch utility (plan 4.6, 8.4).
import type { TagColor } from '$lib/api/types.js';

export const TAG_COLORS = [
	'blue',
	'ochre',
	'green',
	'violet',
	'rose',
	'neutral',
	'contrast'
] as const;

export const tagSwatchClass: Record<TagColor, string> = {
	blue: 'bg-chart-1',
	ochre: 'bg-chart-2',
	green: 'bg-chart-3',
	violet: 'bg-chart-4',
	rose: 'bg-chart-5',
	neutral: 'bg-muted-foreground',
	contrast: 'bg-foreground'
};

/** Reports whether a value is one of the seven tag colours. */
export function isTagColor(value: unknown): value is TagColor {
	return typeof value === 'string' && (TAG_COLORS as readonly string[]).includes(value);
}
