// Document titles of the Settings tabs (plan 8.1.7): one format, "<tab> – Settings – Kinex Mark".
import { describe, expect, test } from 'vitest';

const pages = import.meta.glob('/src/routes/*/settings/*/+page.svelte', {
	query: '?raw',
	import: 'default',
	eager: true
}) as Record<string, string>;

describe('Settings page titles', () => {
	test('every tab names itself through settings.pageTitle', () => {
		const tabs = Object.keys(pages).map((path) => path.split('/').at(-2));
		expect(tabs.sort()).toEqual(['display', 'privacy', 'profile', 'security']);
		for (const [path, source] of Object.entries(pages)) {
			const tab = path.split('/').at(-2);
			expect(source, path).toContain(
				`<PageTitle title={t('settings.pageTitle', { section: t('nav.settingsTabs.${tab}') })} />`
			);
		}
	});
});
