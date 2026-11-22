// The single tag colour mapping (plan 4.6, 8.4).
import { describe, expect, test } from 'vitest';
import { isTagColor, TAG_COLORS, tagSwatchClass } from './colors.js';

describe('tag colours', () => {
	test('every colour maps to a token background utility', () => {
		expect(Object.keys(tagSwatchClass).sort()).toEqual([...TAG_COLORS].sort());
		for (const color of TAG_COLORS) expect(tagSwatchClass[color]).toMatch(/^bg-[a-z0-9-]+$/);
	});

	test('isTagColor accepts the seven colours only', () => {
		expect(isTagColor('blue')).toBe(true);
		expect(isTagColor('contrast')).toBe(true);
		expect(isTagColor('red')).toBe(false);
		expect(isTagColor(1)).toBe(false);
	});
});
