import adapter from '@sveltejs/adapter-auto';
import { sveltekit } from '@sveltejs/kit/vite';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [
		sveltekit({
			adapter: adapter(),
			preprocess: vitePreprocess()
		})
	],
	preview: {
		// The production client is served inside the compose network as
		// `client`; Vite's DNS-rebinding host check must allow that name.
		allowedHosts: ['client']
	},
	resolve: {
		alias: {
			// The shared package is consumed as TypeScript source, with no
			// build step: Vite compiles it alongside the app.
			$shared: fileURLToPath(new URL('../../packages/shared/src', import.meta.url))
		}
	}
});
