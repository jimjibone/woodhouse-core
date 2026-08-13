import basicSsl from '@vitejs/plugin-basic-ssl';
import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	// basicSsl makes the dev server serve HTTPS on a self-signed cert, which is
	// not optional: the refresh cookie is Secure (newTokenCookie in
	// users/auth_service.go) and WebKit refuses to store a Secure cookie on a
	// plain http://localhost origin. Over HTTP, Safari drops it, the next
	// /api/refresh 401s and the login bounces straight back to /login.
	// Chromium exempts localhost so it hides the problem.
	plugins: [basicSsl(), tailwindcss(), sveltekit()],
	server: {
		proxy: {
			'/api': {
				target: 'https://localhost:4080',
				secure: false // woodhouse-core serves a self-signed cert
			}
		}
	}
});
