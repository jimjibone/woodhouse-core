import { type Subscriber, writable } from 'svelte/store';
import { Device, Service } from '$lib/api/v1/clients/client_service_pb';
import { ConnectError, Code, type PromiseClient } from '@connectrpc/connect';
import { UserService } from '$lib/api/v1/clients/user_service_connect';
import { DevicesStreamRequest } from '$lib/api/v1/clients/user_service_pb';
import { getDeviceName } from '$lib/apitools';
import { UserServiceClient } from './user-service-client';
import { Streamer, type HeartbeatHandler } from './streamer';

export type DeviceStoreType = {
	connected: boolean;
	backoff: number;
	devices: Device[];
};

// We'll use a singleton streamer which will manage reconnections.
// let streamer: Streamer | undefined = undefined;
let streamer: Streamer<typeof UserService> | undefined = undefined;

// Create a writable store.
const { subscribe, set, update } = writable<DeviceStoreType>(
	{ connected: false, backoff: 0, devices: [] },
	(set: Subscriber<DeviceStoreType>) => {
		console.log("devices stream subscriber started");

		if (streamer === undefined) {
			streamer = new Streamer(UserServiceClient, streamDevices, backoffHandler);
		}

		return () => {
			console.log("devices stream subscriber finished");
			if (streamer !== undefined) {
				streamer.stop();
			}
		};
	}
);

export const DeviceStore = {
	subscribe
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

const streamDevices = async (client: PromiseClient<typeof UserService>, abortSignal: AbortSignal, heartbeat: HeartbeatHandler) => {
	let didConnect = false;
	const request = new DevicesStreamRequest({});
	try {
		// console.log("streamDevices: starting stream");
		for await (const response of client.devicesStream(request, { signal: abortSignal })) {
			heartbeat();
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
	}

	update((prev: DeviceStoreType) => {
		prev.connected = false;
		return prev;
	});

	return didConnect;
};

const backoffHandler = (backoff: number) => {
	update((prev: DeviceStoreType) => {
		prev.backoff = backoff;
		return prev;
	});
};
