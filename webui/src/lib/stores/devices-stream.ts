import { type Subscriber, writable } from 'svelte/store';
import { ConnectError, Code, type Client, type CallOptions } from '@connectrpc/connect';
import { create } from "@bufbuild/protobuf";
import { DevicesStreamRequestSchema, UserService } from '$lib/api/v1/clients/user_service_pb';
import { Device_DeviceType, Service_ServiceType, type Device, type Service, type TimeAttribute } from '$lib/api/v1/clients/client_service_pb';
import { UserServiceClient } from './user-service-client';
import { Streamer, type HeartbeatHandler } from './streamer';
import { getAccessToken } from '$lib/stores/auth-store';

export type DevicesStoreType = {
	connected: boolean;
	backoff: number;
	devices: DevicesStoreDevice[];
};

export type DevicesStoreDevice = {
	id: string;
	typ: Device_DeviceType;
	name: string | undefined;
	online: boolean;
	batteryLevel: bigint | undefined;
	lastSeen: TimeAttribute | undefined;
	services: Service[];
};

// We'll use a singleton streamer which will manage reconnections.
// let streamer: Streamer | undefined = undefined;
let streamer: Streamer<typeof UserService> | undefined = undefined;

// Create a writable store.
const { subscribe, set, update } = writable<DevicesStoreType>(
	{ connected: false, backoff: 0, devices: [] },
	(set: Subscriber<DevicesStoreType>) => {
		console.log("devices stream subscriber started");

		if (streamer === undefined) {
			streamer = new Streamer("devices", UserServiceClient, streamDevices, backoffHandler);
		} else {
			streamer.restart();
		}

		return () => {
			console.log("devices stream subscriber finished");
			if (streamer !== undefined) {
				streamer.stop();
			}
		};
	}
);

export const DevicesStore = {
	subscribe
};

const createDevice = (next: Device): DevicesStoreDevice => {
	let prev : DevicesStoreDevice = {
		id: next.id,
		typ: next.typ,
		name: next.id,
		online: false,
		lastSeen: undefined,
		batteryLevel: undefined,
		services: next.services
	};
	return updateDevice(prev, next);
};

const updateDevice = (prev: DevicesStoreDevice, next: Device): DevicesStoreDevice => {
	prev.typ = next.typ;
	if (next.fullState) {
		// Remove all services as we're about to receive the complete new set.
		prev.name = next.id;
		prev.online = false;
		prev.lastSeen = undefined;
		prev.batteryLevel = undefined;
		prev.services = [];
	}
	for (let i = 0; i < next.services.length; i++) {
		if (next.services[i].typ === Service_ServiceType.INFO) {
			for (const attr of next.services[i].attrs) {
				if (attr.id === "name") {
					prev.name = attr.text!.value;
					break;
				}
			}
		}
		if (next.services[i].typ === Service_ServiceType.ONLINE) {
			for (const attr of next.services[i].attrs) {
				if (attr.id === "online") {
					prev.online = attr.bool!.value;
				} else if (attr.id === "last_seen") {
					prev.lastSeen = attr.time;
				}
			}
		}
		if (next.services[i].typ === Service_ServiceType.BATTERY) {
			for (const attr of next.services[i].attrs) {
				if (attr.id === "level") {
					prev.batteryLevel = attr.int!.value;
					break;
				}
			}
		}

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

const streamDevices = async (client: Client<typeof UserService>, abortSignal: AbortSignal, heartbeat: HeartbeatHandler) => {
	let didConnect = false;
	const request = create(DevicesStreamRequestSchema, {});
	try {
		// console.log("streamDevices: starting stream");
		const options: CallOptions = {
			signal: abortSignal,
			headers: { "authorization": getAccessToken() }
		};
		for await (const response of client.devicesStream(request, options)) {
			heartbeat();
			didConnect = true;

			// All fields will be empty if this is a keepalive message.
			if (response.id !== "") {
				// console.log("streamDevices: update: " + response.toJsonString());
				update((prev: DevicesStoreType) => {
					let foundDeviceService = false;
					for (let d = 0; d < prev.devices.length; d++) {
						if (prev.devices[d].id === response.id) {
							// console.log("streamDevices: update: found " + response.deviceService?.key);
							foundDeviceService = true;
							prev.devices[d] = updateDevice(prev.devices[d], response);
							break;
						}
					}
					if (!foundDeviceService) {
						// console.log("streamDevices: update: not found " + response.id);
						prev.devices = [...prev.devices, createDevice(response)];
					}

					prev.devices = prev.devices.sort((a, b) => {
						const aName = a.name ? a.name : a.id;
						const bName = b.name ? b.name : b.id;
						return aName > bName ? 1 : bName > aName ? -1 : 0;
					});

					prev.connected = true;

					return prev;
				});
			} else {
				update((prev: DevicesStoreType) => {
					prev.connected = true;
					return prev;
				});
			}
		}
	} catch (err) {
		if (err instanceof ConnectError) {
			if (err.code !== Code.Unknown && err.code !== Code.Canceled) {
				console.error('streamDevices: error stream: (' + err.code + ') ' + err.message);
			}
		}
	}

	update((prev: DevicesStoreType) => {
		prev.connected = false;
		return prev;
	});

	return didConnect;
};

const backoffHandler = (backoff: number) => {
	update((prev: DevicesStoreType) => {
		prev.backoff = backoff;
		return prev;
	});
};
