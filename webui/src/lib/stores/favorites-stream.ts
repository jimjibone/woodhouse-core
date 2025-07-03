import { type Subscriber, writable } from 'svelte/store';
import { ConnectError, Code, type Client } from '@connectrpc/connect';
import { create, toJsonString } from "@bufbuild/protobuf";
import { FavoritesStreamResponseSchema, UserService } from '$lib/api/v1/clients/user_service_pb';
import { type DeviceService, FavoritesStreamRequestSchema } from '$lib/api/v1/clients/user_service_pb';
import { UserServiceClient } from './user-service-client';
import { Streamer, type HeartbeatHandler } from './streamer';

export type FavoritesStoreType = {
	connected: boolean;
	backoff: number;
	deviceServices: DeviceService[];
};

// We'll use a singleton streamer which will manage reconnections.
// let streamer: Streamer | undefined = undefined;
let streamer: Streamer<typeof UserService> | undefined = undefined;

// Create a writable store.
const { subscribe, set, update } = writable<FavoritesStoreType>(
	{ connected: false, backoff: 0, deviceServices: [] },
	(set: Subscriber<FavoritesStoreType>) => {
		console.log("favorites stream subscriber started");

		if (streamer === undefined) {
			streamer = new Streamer(UserServiceClient, streamFavorites, backoffHandler);
		}

		return () => {
			console.log("favorites stream subscriber finished");
			if (streamer !== undefined) {
				streamer.stop();
			}
		};
	}
);

export const FavoritesStore = {
	subscribe
};

const updateDeviceService = (prev: DeviceService, next: DeviceService): DeviceService => {
	// No merging required at the moment...
	return next;
};

const getDeviceServiceName = (ds: DeviceService): string => {
	if (ds.hasDeviceName && ds.service !== undefined) {
		if (ds.service.alias!=='') {
			return ds.deviceName+": "+ds.service.alias
		}
		return ds.deviceName
	}
	return ds.key
};

const streamFavorites = async (client: Client<typeof UserService>, abortSignal: AbortSignal, heartbeat: HeartbeatHandler) => {
	let didConnect = false;
	const request = create(FavoritesStreamRequestSchema, {});
	try {
		// console.log("streamFavorites: starting stream");
		for await (const response of client.favoritesStream(request, { signal: abortSignal })) {
			heartbeat();
			didConnect = true;

			// All fields will be empty if this is a keepalive message.
			if (response.deviceService !== undefined) {
				// console.log("streamFavorites: update: " + response.toJsonString());
				update((prev: FavoritesStoreType) => {
					let foundDeviceService = false;
					for (let d = 0; d < prev.deviceServices.length; d++) {
						if (prev.deviceServices[d].key === response.deviceService?.key) {
							// console.log("streamFavorites: update: found " + response.deviceService?.key);
							foundDeviceService = true;
							prev.deviceServices[d] = updateDeviceService(prev.deviceServices[d], response.deviceService);
							break;
						}
					}
					if (!foundDeviceService) {
						// console.log("streamDevices: update: not found " + response.id);
						prev.deviceServices = [...prev.deviceServices, response.deviceService!];
					}

					prev.deviceServices = prev.deviceServices.sort((a, b) => {
						const aName = getDeviceServiceName(a);
						const bName = getDeviceServiceName(b);
						return aName > bName ? 1 : bName > aName ? -1 : 0;
					});

					prev.connected = true;

					return prev;
				});
			} else if (response.keyRemoved !== '') {
				console.log("streamFavorites: removed: " + toJsonString(FavoritesStreamResponseSchema, response));
				update((prev: FavoritesStoreType) => {
					for (let d = 0; d < prev.deviceServices.length; d++) {
						if (prev.deviceServices[d].key === response.keyRemoved) {
							console.log("streamFavorites: removed: found " + response.keyRemoved);
							prev.deviceServices.splice(d, 1);
							break;
						}
					}

					prev.connected = true;

					return prev;
				});
			} else {
				update((prev: FavoritesStoreType) => {
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

	update((prev: FavoritesStoreType) => {
		prev.connected = false;
		return prev;
	});

	return didConnect;
};

const backoffHandler = (backoff: number) => {
	update((prev: FavoritesStoreType) => {
		prev.backoff = backoff;
		return prev;
	});
};
