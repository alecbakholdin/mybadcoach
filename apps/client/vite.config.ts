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
		// `changeOrigin: false` preserves the browser's Host header so the
		// server's same-origin CSRF check (Origin vs Host) still matches; the
		// string shorthand would rewrite Host to the target and reject every
		// mutating RPC.
		proxy: {
			'/mybadcoach': { target: 'http://localhost:8080', changeOrigin: false }
		}
	},
	preview: {
		// The production client is served inside the compose network as
		// `client`; Vite's DNS-rebinding host check must allow that name.
		allowedHosts: ['client'],
		// Frontend-only integration seam: the SPA's Connect transport stays
		// same-origin (`baseUrl: '/'`), so preview forwards RPCs to the stub
		// Connect server by default; the compose stack overrides the target via
		// MBC_PROXY_TARGET to reach the real Go server. The Host
		// header is preserved for the same CSRF reason as `server.proxy`.
		proxy: {
			'/mybadcoach': { target: process.env.MBC_PROXY_TARGET ?? 'http://127.0.0.1:4174', changeOrigin: false }
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
