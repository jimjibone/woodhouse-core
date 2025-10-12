import { doRefresh } from '$lib/stores/auth-store';

export async function load() {
	// Wait until first token refresh is complete (doesn't have to succeed).
	await doRefresh();
}
