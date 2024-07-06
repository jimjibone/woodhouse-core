import { type Subscriber, writable } from 'svelte/store';
import {
	ActionRequest,
	Device,
	Service,
	Value
} from '$lib/api/v1/clients/client_service_pb';

import { createGrpcWebTransport } from '@connectrpc/connect-web';
import { createPromiseClient, ConnectError, Code, type PromiseClient } from '@connectrpc/connect';
import { UserService } from '$lib/api/v1/clients/user_service_connect';
import { DevicesStreamRequest } from '$lib/api/v1/clients/user_service_pb';
import { getDeviceName } from '$lib/apitools';

export type DeviceStoreType = {
	connected: boolean;
	backoff: number;
	devices: Device[];
};

// Create the GRPC-Web transport and client.
const transport = createGrpcWebTransport({
	baseUrl: '/api'
});
const client = createPromiseClient(UserService, transport);

// We'll use a singleton streamer which will manage reconnections.
let streamer: Streamer | undefined = undefined;

// Create a writable store.
const { subscribe, set, update } = writable<DeviceStoreType>(
	{ connected: false, backoff: 0, devices: [] },
	(set: Subscriber<DeviceStoreType>) => {
		// console.log("subscriber started");

		if (streamer === undefined) {
			streamer = new Streamer(client);
		}

		return () => {
			// TODO: close stream when all subscribers stop.
			// console.log("subscriber finished");
		};
	}
);

export const DeviceStore = {
	subscribe
};

export const DeviceAction = async (deviceID: string, serviceID: string, vals: Value[]) => {
	const request = new ActionRequest({
		deviceId: deviceID,
		serviceId: serviceID,
		values: vals
	});
	console.log('sending action: ' + request.toJsonString());
	try {
		for await (const response of client.sendAction(request)) {
			console.log('received action: ' + response.toJsonString());
			// responses = [...responses, response];
		}
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error action: ' + err.message);
		}
	}
};

const updateDevice = (prev: Device, next: Device): Device => {
	prev.typ = next.typ;
	if (next.fullState) {
		// Remove all services as we're about to receive the complete new set.
		prev.services = [];
	}
	for (let i = 0; i < next.services.length; i++) {
		let foundService = false;
		for (let j = 0; j < prev.services.length; j++) {
			if (next.services[i].id === prev.services[j].id) {
				foundService = true;
				prev.services[j] = updateService(prev.services[j], next.services[i]);
				break;
			}
		}
		if (!foundService) {
			if (!foundService) prev.services = [...prev.services, next.services[i]];
		}
	}
	return prev;
};

const updateService = (prev: Service, next: Service): Service => {
	prev.typ = next.typ;
	prev.alias = next.alias;
	for (let i = 0; i < next.attrs.length; i++) {
		let foundAttr = false;
		for (let j = 0; j < prev.attrs.length; j++) {
			if (next.attrs[i].id === prev.attrs[j].id) {
				foundAttr = true;
				prev.attrs[j] = next.attrs[i];
				break;
			}
		}
		if (!foundAttr) {
			if (!foundAttr) prev.attrs = [...prev.attrs, next.attrs[i]];
		}
	}
	return prev;
};

const streamDevices = async (
	client: PromiseClient<typeof UserService>,
	onFinish: (resetBackoff: boolean) => void
) => {
	// Use a timer to periodically check if the connection has died. If it
	// has then trigger the abort controller to cancel the stream and then
	// fire up another one after a backoff delay.
	let lastrx = Date.now();
	let didConnect = false;
	const controller = new AbortController();
	const interval = setInterval(() => {
		const now = Date.now();
		if (now - lastrx > 11000) {
			controller.abort();
		}
	}, 1000);
	const request = new DevicesStreamRequest({});
	try {
		// console.log("streamDevices: starting stream");
		for await (const response of client.devicesStream(request, { signal: controller.signal })) {
			lastrx = Date.now();
			didConnect = true;

			// ID will be empty if this is a keepalive message.
			if (response.id !== '') {
				// console.log("streamDevices: device: " + response.toJsonString());
				update((prev: DeviceStoreType) => {
					let foundDevice = false;
					for (let d = 0; d < prev.devices.length; d++) {
						if (prev.devices[d].id === response.id) {
							// console.log("streamDevices: update: found " + response.id);
							foundDevice = true;
							prev.devices[d] = updateDevice(prev.devices[d], response);
							break;
						}
					}
					if (!foundDevice) {
						// console.log("streamDevices: update: not found " + response.id);
						prev.devices = [...prev.devices, response];
					}

					prev.devices = prev.devices.sort((a, b) => {
						const aName = getDeviceName(a);
						const bName = getDeviceName(b);
						return aName > bName ? 1 : bName > aName ? -1 : 0;
					});

					prev.connected = true;

					return prev;
				});
			} else {
				update((prev: DeviceStoreType) => {
					prev.connected = true;
					return prev;
				});
			}
		}
	} catch (err) {
		if (err instanceof ConnectError) {
			if (err.code !== Code.Unknown) {
				console.error('streamDevices: error stream: (' + err.code + ') ' + err.message);
			}
		}

		clearInterval(interval);

		update((prev: DeviceStoreType) => {
			prev.connected = false;
			return prev;
		});

		const resetBackoff = didConnect;
		onFinish(resetBackoff);
	}
};

class Streamer {
	constructor(client: PromiseClient<typeof UserService>) {
		this.retry(client);
	}

	backoff = 1000;
	minBackoff = 1000;
	maxBackoff = 4000;

	retry = (client: PromiseClient<typeof UserService>) => {
		streamDevices(client, (resetBackoff: boolean) => {
			if (resetBackoff) {
				this.backoff = 0;
			} else {
				if (this.backoff === 0) {
					this.backoff = this.minBackoff;
				} else {
					this.backoff = this.backoff * 2;
					if (this.backoff > this.maxBackoff) {
						this.backoff = this.maxBackoff;
					}
				}
			}
			update((prev: DeviceStoreType) => {
				prev.backoff = this.backoff;
				return prev;
			});
			setTimeout(() => {
				this.retry(client);
			}, this.backoff);
		});
	};
}
