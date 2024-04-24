import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		// Use the static adapter along with 'ssr = false' in +layout.js to
		// generate a single page app.
		adapter: adapter({
			fallback: "index.html"
		}),

		alias: {
			"@/*": "./src/lib/*",
		},
	},

	preprocess: [vitePreprocess({})]
};

export default config;
