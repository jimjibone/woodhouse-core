import { setContext, getContext } from 'svelte';

const KEY = Symbol('connection-status');

export class ConnectionStatus {
	connected = $state(false);
	reconnecting = $state(false);

	set(connected: boolean, reconnecting: boolean) {
		this.connected = connected;
		this.reconnecting = reconnecting;
	}

	reset() {
		this.connected = false;
		this.reconnecting = false;
	}
}

export function createConnectionContext(): ConnectionStatus {
	const status = new ConnectionStatus();
	setContext(KEY, status);
	return status;
}

export function useConnectionContext(): ConnectionStatus {
	return getContext<ConnectionStatus>(KEY);
}
