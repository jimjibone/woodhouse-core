import { writable } from 'svelte/store';
import { ConnectError } from '@connectrpc/connect';
import { GetSettings, type ServerSettings } from './requests';

// PRODUCT_NAME is what the interface shows unless an admin opts into showing
// the instance name instead.
export const PRODUCT_NAME = 'Woodhouse';

export type SettingsState = ServerSettings & {
	loaded: boolean;
};

// Settings are a handful of scalars changed by one admin on a settings page,
// so unlike devices or clients they do not warrant a stream. This is fetched
// once and updated in place when the user saves.
const settingsValue = writable<SettingsState>({
	instanceName: '',
	showInstanceName: false,
	loaded: false
});

let inflight: Promise<void> | null = null;

export const settings = {
	subscribe: settingsValue.subscribe
};

// displayName is what the sidebar and the tab title should show: the product
// name by default, the instance name only when an admin has opted in.
export function displayName(state: SettingsState): string {
	if (state.showInstanceName && state.instanceName !== '') {
		return state.instanceName;
	}
	return PRODUCT_NAME;
}

// loadSettings fetches the settings once. Concurrent callers share the same
// request, and it is a no-op once loaded unless force is set.
export async function loadSettings(force = false): Promise<void> {
	if (inflight) return inflight;

	let alreadyLoaded = false;
	const unsubscribe = settingsValue.subscribe((value) => (alreadyLoaded = value.loaded));
	unsubscribe();
	if (alreadyLoaded && !force) return;

	inflight = (async () => {
		const result = await GetSettings();
		if (result instanceof ConnectError) {
			// Leave loaded false so a later attempt can retry. The interface
			// falls back to the product name until this succeeds.
			console.error('failed to load settings: ' + result.message);
			return;
		}
		settingsValue.set({ ...result, loaded: true });
	})();

	try {
		await inflight;
	} finally {
		inflight = null;
	}
}

// applySettings records settings the server has already accepted and stored.
export function applySettings(stored: ServerSettings) {
	settingsValue.set({ ...stored, loaded: true });
}
