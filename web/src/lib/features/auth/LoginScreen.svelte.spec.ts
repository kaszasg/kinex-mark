import { fireEvent, screen, waitFor, within } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { goto } from '$app/navigation';
import { i18n, t } from '$lib/i18n/index.js';
import { publicConfig } from '$lib/state/public-config.svelte.js';
import { session } from '$lib/state/session.svelte.js';
import { makePublicConfig } from '$lib/test/fixtures.js';
import { json, mockApi } from '$lib/test/mock-api.js';
import { renderWithApp } from '$lib/test/render.js';
import LoginScreen from './LoginScreen.svelte';

beforeEach(() => {
	vi.mocked(goto).mockClear();
	publicConfig.config = null;
});

afterEach(() => {
	vi.unstubAllGlobals();
	session.clear();
	publicConfig.config = null;
});

const FEATURES = ['capture', 'tags', 'filter', 'ranking', 'expiry'] as const;

describe('LoginScreen', () => {
	test('shows the brand surface, the hero, the features and the sign-in form', () => {
		mockApi();
		renderWithApp(LoginScreen, {});
		expect(document.title).toBe(`${t('auth.login.title')} – Kinex Mark`);
		expect(screen.getByRole('main')).toHaveAttribute('id', 'main');
		expect(screen.getByRole('heading', { level: 1, name: t('auth.login.title') })).toBeVisible();
		expect(screen.getByRole('heading', { level: 2, name: t('landing.headline') })).toBeVisible();
		expect(screen.getByText(t('landing.intro'))).toBeInTheDocument();
		const image = screen.getByRole('img', { name: t('landing.hero.alt') });
		expect(image).toHaveAttribute('src', '/images/hero.webp');
		expect(image).toHaveAttribute('width', '1600');
		expect(image).toHaveAttribute('height', '900');
		const features = screen.getByRole('region', { name: t('landing.features.label') });
		const items = within(features).getAllByRole('listitem');
		expect(items).toHaveLength(FEATURES.length);
		FEATURES.forEach((key, index) => {
			expect(items[index]).toHaveTextContent(t(`landing.features.${key}.title`));
			expect(items[index]).toHaveTextContent(t(`landing.features.${key}.text`));
		});
	});

	test('the footer holds the language switch, the theme toggle and the privacy link', () => {
		mockApi();
		renderWithApp(LoginScreen, {});
		const footer = screen.getByRole('contentinfo', { name: t('auth.login.footer') });
		expect(within(footer).getByRole('group', { name: t('nav.locale.label') })).toBeVisible();
		expect(within(footer).getByRole('button', { name: t('nav.theme.toggle') })).toBeVisible();
		expect(within(footer).getByRole('link', { name: t('nav.privacyNotice') })).toHaveAttribute(
			'href',
			'/privacy'
		);
	});

	test('the version appears once the public configuration has loaded', async () => {
		const api = mockApi({
			'GET /api/public/config': json(makePublicConfig({ version: '1.4.2' }))
		});
		renderWithApp(LoginScreen, {});
		expect(document.querySelector('[data-version]')).toBeNull();
		expect(await screen.findByText(t('auth.login.version', { version: '1.4.2' }))).toHaveClass(
			'font-mono'
		);
		expect(api.calls('GET /api/public/config')).toHaveLength(1);
	});

	test('without a configuration the version stays hidden', async () => {
		const api = mockApi();
		renderWithApp(LoginScreen, {});
		await waitFor(() => expect(api.calls('GET /api/public/config')).toHaveLength(1));
		expect(document.querySelector('[data-version]')).toBeNull();
	});

	test('the hero fades in on load and a missing file leaves the empty panel', async () => {
		mockApi();
		renderWithApp(LoginScreen, {});
		const frame = document.querySelector('[data-hero]');
		const image = screen.getByRole('img', { name: t('landing.hero.alt') });
		expect(frame).toHaveAttribute('data-hero', 'loading');
		expect(image).toHaveClass('opacity-0');
		await fireEvent.load(image);
		expect(frame).toHaveAttribute('data-hero', 'loaded');
		expect(image).toHaveClass('opacity-100');
		await fireEvent.error(image);
		expect(frame).toHaveAttribute('data-hero', 'failed');
		expect(screen.queryByRole('img', { name: t('landing.hero.alt') })).toBeNull();
	});

	test('a cached hero image counts as loaded at once', async () => {
		mockApi();
		const complete = vi.spyOn(HTMLImageElement.prototype, 'complete', 'get').mockReturnValue(true);
		const width = vi.spyOn(HTMLImageElement.prototype, 'naturalWidth', 'get').mockReturnValue(1600);
		renderWithApp(LoginScreen, {});
		await waitFor(() =>
			expect(document.querySelector('[data-hero]')).toHaveAttribute('data-hero', 'loaded')
		);
		expect(complete).toHaveBeenCalled();
		expect(width).toHaveBeenCalled();
	});

	test('from 1024 px the form column is one viewport high, so the footer stays in view', () => {
		mockApi();
		renderWithApp(LoginScreen, {});
		const column = document.querySelector('[data-form-column]')!;
		expect(column).toHaveClass('lg:sticky', 'lg:top-0', 'lg:h-dvh', 'lg:self-start');
		expect(column.querySelector('footer')).toHaveClass('mt-10');
		expect(column.lastElementChild?.tagName).toBe('FOOTER');
	});

	test('768-1023 px: the brand card sits behind the logo, the headline and the intro', () => {
		mockApi();
		const { container } = renderWithApp(LoginScreen, {});
		const card = container.querySelector('.kx-panel.kx-panel-lg[aria-hidden="true"]')!;
		expect(card).toHaveClass('md:block', 'lg:hidden');
		// kx-panel is position: relative; only a negative z-index inside an isolated grid keeps
		// it under the static header and hero section that share its grid cells.
		expect(card).toHaveClass('-z-10');
		expect(card.parentElement).toHaveClass('isolate');
	});

	test('passes the reason to the form', () => {
		mockApi();
		renderWithApp(LoginScreen, { reason: 'expired' });
		expect(screen.getByText(t('auth.expired.title'))).toBeInTheDocument();
	});

	test('follows the language switch', async () => {
		mockApi();
		renderWithApp(LoginScreen, {});
		i18n.setLocale('en', { persist: false });
		expect(await screen.findByRole('heading', { name: 'Links, ranked by use.' })).toBeVisible();
		expect(screen.getByRole('img', { name: /^Line diagram/ })).toBeInTheDocument();
	});
});
