import { fileURLToPath } from 'node:url';

import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [
		sveltekit({
			adapter: adapter({
				// SPA: every client-side route falls back to the index.html shell.
				fallback: 'index.html'
			}),
			preprocess: vitePreprocess()
		})
	],
	server: {
		// Dev: forward Connect-RPC calls to the Go server on its default port.
		proxy: {
			'/mybadcoach': 'http://localhost:8080'
		}
	},
	preview: {
		// The production client is served inside the compose network as
		// `client`; Vite's DNS-rebinding host check must allow that name.
		allowedHosts: ['client'],
		// Frontend-only integration seam: the SPA's Connect transport stays
		// same-origin (`baseUrl: '/'`), so preview forwards RPCs to the stub
		// Connect server, just as dev forwards them to the Go server.
		proxy: {
			'/mybadcoach': 'http://127.0.0.1:4174'
		}
	},
	resolve: {
		alias: {
			// The shared package is consumed as TypeScript source, with no
			// build step: Vite compiles it alongside the app.
			$shared: fileURLToPath(new URL('../../packages/shared/src', import.meta.url))
		}
	}
});
