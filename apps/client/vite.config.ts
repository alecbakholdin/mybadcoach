import adapter from '@sveltejs/adapter-auto';
import { sveltekit } from '@sveltejs/kit/vite';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [
		sveltekit({
			adapter: adapter(),
			preprocess: vitePreprocess(),
			alias: {
				// Resolved relative to apps/client by SvelteKit. The shared
				// package is consumed as TypeScript source, with no build step:
				// Vite compiles it alongside the app.
				$shared: '../../packages/shared/src'
			}
		})
	]
});
