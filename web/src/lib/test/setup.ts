// Global Vitest setup for the jsdom environment (plan 9.3.1).
import '@testing-library/jest-dom/vitest';
import { beforeEach, vi } from 'vitest';
import { i18n } from '$lib/i18n/index.js';
import { setTestPage } from './app-state.svelte.js';

// SvelteKit's router does not run in unit tests: navigation calls are recorded instead.
vi.mock('$app/navigation', () => ({
	goto: vi.fn(async () => {}),
	afterNavigate: vi.fn(),
	beforeNavigate: vi.fn(),
	onNavigate: vi.fn(),
	invalidate: vi.fn(async () => {}),
	invalidateAll: vi.fn(async () => {}),
	preloadData: vi.fn(async () => ({ type: 'loaded', status: 200, data: {} })),
	preloadCode: vi.fn(async () => {}),
	pushState: vi.fn(),
	replaceState: vi.fn(),
	disableScrollHandling: vi.fn()
}));

// $app/state is replaced by a reactive test page that tests point at a path.
vi.mock('$app/state', async () => {
	const mod = await import('./app-state.svelte.js');
	return { page: mod.page, navigating: mod.navigating, updated: mod.updated };
});

// jsdom has no matchMedia; mode-watcher, the sidebar and the mobile hook query it.
if (typeof window.matchMedia !== 'function') {
	Object.defineProperty(window, 'matchMedia', {
		configurable: true,
		writable: true,
		value: (query: string): MediaQueryList => {
			const list = {
				matches: false,
				media: query,
				onchange: null,
				addListener: () => {},
				removeListener: () => {},
				addEventListener: () => {},
				removeEventListener: () => {},
				dispatchEvent: () => false
			};
			return list as MediaQueryList;
		}
	});
}

// jsdom has no IntersectionObserver; observed elements are simply never reported as visible.
class StubIntersectionObserver implements IntersectionObserver {
	readonly root: Element | Document | null = null;
	readonly rootMargin = '0px';
	readonly scrollMargin = '0px';
	readonly thresholds: ReadonlyArray<number> = [0];
	observe(): void {}
	unobserve(): void {}
	disconnect(): void {}
	takeRecords(): IntersectionObserverEntry[] {
		return [];
	}
}

if (typeof window.IntersectionObserver !== 'function') {
	Object.defineProperty(window, 'IntersectionObserver', {
		configurable: true,
		writable: true,
		value: StubIntersectionObserver
	});
}

// jsdom has no ResizeObserver; floating UI (popover, tooltip) only needs the methods.
if (typeof window.ResizeObserver !== 'function') {
	class StubResizeObserver {
		observe(): void {}
		unobserve(): void {}
		disconnect(): void {}
	}
	Object.defineProperty(window, 'ResizeObserver', {
		configurable: true,
		writable: true,
		value: StubResizeObserver
	});
}

// jsdom lacks scrollIntoView and the pointer-capture API used by bits-ui.
if (!Element.prototype.scrollIntoView) Element.prototype.scrollIntoView = () => {};
if (!Element.prototype.hasPointerCapture) Element.prototype.hasPointerCapture = () => false;
if (!Element.prototype.releasePointerCapture) Element.prototype.releasePointerCapture = () => {};

// Clipboard mock: tests read what was written through navigator.clipboard.
let clipboardText = '';
Object.defineProperty(navigator, 'clipboard', {
	configurable: true,
	value: {
		writeText: vi.fn(async (text: string) => {
			clipboardText = text;
		}),
		readText: vi.fn(async () => clipboardText)
	}
});

beforeEach(() => {
	clipboardText = '';
	// Every test starts in Hungarian, without a hand-picked locale, on the root page.
	i18n.setLocale('hu', { persist: false });
	i18n.clearExplicit();
	setTestPage('/');
});
