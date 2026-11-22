import { describe, expect, it } from 'vitest';
import { foldText } from './fold.js';

describe('foldText', () => {
	it('drops case and accents like the server key', () => {
		expect(foldText('Árvíztűrő Tükörfúrógép')).toBe('arvizturo tukorfurogep');
		expect(foldText('Crème Brûlée')).toBe('creme brulee');
		expect(foldText('ABC')).toBe('abc');
		expect(foldText('')).toBe('');
	});

	it('keeps letters without a decomposition and recomposes other scripts', () => {
		expect(foldText('Ősz Øre Straße')).toBe('osz øre straße');
		expect(foldText('한국어')).toBe('한국어');
	});
});
