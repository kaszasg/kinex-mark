import { defineConfig, devices } from '@playwright/test';

// E2E suite against the real binary (plan 9.4.1). start-server.mjs starts the server and the
// metadata fixture origin; global setup seeds the users and saves their signed-in states.
export default defineConfig({
	testDir: 'e2e/specs',
	globalSetup: './e2e/support/global-setup.ts',
	globalTeardown: './e2e/support/global-teardown.ts',
	timeout: 90_000,
	fullyParallel: false,
	workers: 1,
	retries: process.env.CI ? 1 : 0,
	reporter: [['list'], ['html', { open: 'never' }]],
	use: {
		baseURL: 'http://127.0.0.1:18080',
		locale: 'hu-HU',
		timezoneId: 'Europe/Budapest',
		colorScheme: 'dark',
		viewport: { width: 1440, height: 900 },
		trace: 'retain-on-failure',
		screenshot: 'only-on-failure'
	},
	projects: [
		{
			name: 'chromium',
			use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 900 } },
			testIgnore: [/responsive\.spec\.ts/, /\.last\.spec\.ts/]
		},
		{
			name: 'mobile',
			use: { ...devices['Pixel 7'] },
			testMatch: /responsive\.spec\.ts/,
			dependencies: ['chromium']
		},
		// Specs that bump the privacy notice version run last, after every other project.
		{
			name: 'chromium-last',
			use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 900 } },
			testMatch: /\.last\.spec\.ts/,
			dependencies: ['chromium', 'mobile']
		}
	],
	webServer: {
		command: 'node e2e/support/start-server.mjs',
		url: 'http://127.0.0.1:18080/healthz',
		// Must stay false, even though Playwright's "is already used" error suggests otherwise:
		// 127.0.0.1:18080 and 18081 belong to this harness alone (plan 3.4). A server found there
		// was started by someone else (manual GUI checks use 8080, plan 9.5 and 11.8), so its data
		// and admin password are unknown. Stop that server instead of reusing it.
		reuseExistingServer: false,
		timeout: 30_000,
		stdout: 'pipe',
		gracefulShutdown: { signal: 'SIGTERM', timeout: 5_000 }
	}
});
