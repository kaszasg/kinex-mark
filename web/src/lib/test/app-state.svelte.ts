// Reactive stand-in for $app/state in component tests: set page.url before rendering.

class TestPage {
	url = $state(new URL('http://localhost/'));
	params = $state<Record<string, string>>({});
	route = $state<{ id: string | null }>({ id: null });
	status = $state(200);
	error = $state<App.Error | null>(null);
	data = $state<Record<string, unknown>>({});
	form = $state<unknown>(null);
	state = $state<App.PageState>({});
}

export const page = new TestPage();

/** Points the test page at a path (and optional query), resetting the error state. */
export function setTestPage(path: string, status = 200, error: App.Error | null = null): void {
	page.url = new URL(path, 'http://localhost');
	page.status = status;
	page.error = error;
}

export const navigating = { current: null };
export const updated = { current: false, check: async () => false };
