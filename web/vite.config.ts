import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

// The Go server listens here during development (build.ps1 dev, go run ./cmd/kinex-mark).
const backend = 'http://127.0.0.1:8080';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			compilerOptions: {
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},
			// index.html is the shared entry point of every client-side route.
			adapter: adapter({ fallback: 'index.html' }),
			// shadcn-svelte generates $lib imports.
			alias: { $lib: 'src/lib' }
		})
	],
	server: {
		// No changeOrigin: the Go CSRF check compares Origin with the forwarded Host.
		proxy: { '/api': backend, '/healthz': backend }
	}
});
