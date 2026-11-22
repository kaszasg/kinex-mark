import { svelteTesting } from '@testing-library/svelte/vite';
import { defineConfig, mergeConfig } from 'vitest/config';
import viteConfig from './vite.config.ts';

export default mergeConfig(
	viteConfig,
	defineConfig({
		plugins: [svelteTesting()],
		test: {
			environment: 'jsdom',
			include: ['src/**/*.{test,spec}.{js,ts}'],
			setupFiles: ['./src/lib/test/setup.ts'],
			expect: { requireAssertions: true },
			restoreMocks: true,
			coverage: {
				provider: 'v8',
				include: ['src/lib/**/*.{ts,svelte}'],
				exclude: [
					'src/lib/components/ui/**',
					'src/lib/hooks/**',
					'src/lib/test/**',
					'src/lib/i18n/messages/**',
					'src/lib/api/types.ts',
					'**/*.{test,spec}.ts',
					'**/*.d.ts'
				],
				reporter: ['text-summary', 'json-summary', 'lcov', 'html'],
				reportsDirectory: './coverage',
				thresholds: { statements: 85, branches: 80, functions: 85, lines: 85 }
			}
		}
	})
);
