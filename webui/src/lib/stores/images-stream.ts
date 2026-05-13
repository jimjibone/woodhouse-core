import { type Subscriber, writable } from 'svelte/store';
import { ConnectError, Code, type Client, type CallOptions } from '@connectrpc/connect';
import { create } from '@bufbuild/protobuf';
import {
	ImagesStreamRequestSchema,
	UserService
} from '$lib/api/v1/clients/user_service_pb';
import { UserServiceClient } from './user-service-client';
import { Streamer, type HeartbeatHandler } from './streamer';
import { getAccessToken } from '$lib/stores/auth-store';

export type CachedImage = {
	deviceId: string;
	serviceId: string;
	attributeId: string;
	/** Object URL created from image data — revoke the previous one when updating. */
	url: string;
	mimeType: string;
	fetchedAt: Date;
};

export type ImagesStoreType = {
	connected: boolean;
	backoff: number;
	/** Keyed by `"deviceId:serviceId:attributeId"` */
	images: Map<string, CachedImage>;
};

let streamer: Streamer<typeof UserService> | undefined = undefined;

const { subscribe, set, update } = writable<ImagesStoreType>(
	{ connected: false, backoff: 0, images: new Map() },
	(set: Subscriber<ImagesStoreType>) => {
		console.log('images stream subscriber started');

		if (streamer === undefined) {
			streamer = new Streamer('images', UserServiceClient, streamImages, backoffHandler);
		} else {
			streamer.restart();
		}

		return () => {
			console.log('images stream subscriber finished');
			if (streamer !== undefined) {
				streamer.stop();
			}
		};
	}
);

export const ImagesStore = {
	subscribe
};

const streamImages = async (
	client: Client<typeof UserService>,
	abortSignal: AbortSignal,
	heartbeat: HeartbeatHandler
) => {
	let didConnect = false;

	const request = create(ImagesStreamRequestSchema, {});
	try {
		const options: CallOptions = {
			signal: abortSignal,
			headers: { authorization: getAccessToken() }
		};

		let gotSnapshot = false;
		const snapshotKeys: string[] = [];

		for await (const response of client.imagesStream(request, options)) {
			heartbeat();
			didConnect = true;

			// Empty response = heartbeat or end-of-snapshot marker.
			if (!response.deviceId) {
				if (!gotSnapshot) {
					gotSnapshot = true;
					// Remove stale images that were not in the snapshot.
					update((prev) => {
						for (const key of prev.images.keys()) {
							if (!snapshotKeys.includes(key)) {
								const old = prev.images.get(key);
								if (old) URL.revokeObjectURL(old.url);
								prev.images.delete(key);
							}
						}
						prev.connected = true;
						return prev;
					});
				} else {
					update((prev) => {
						prev.connected = true;
						return prev;
					});
				}
				continue;
			}

			const key = `${response.deviceId}:${response.serviceId}:${response.attributeId}`;

			if (!gotSnapshot) {
				snapshotKeys.push(key);
			}

			if (response.data && response.data.length > 0) {
				const blob = new Blob([new Uint8Array(response.data)], {
					type: response.mimeType || 'image/jpeg'
				});
				const url = URL.createObjectURL(blob);

				update((prev) => {
					const old = prev.images.get(key);
					if (old) URL.revokeObjectURL(old.url);

					prev.images.set(key, {
						deviceId: response.deviceId,
						serviceId: response.serviceId,
						attributeId: response.attributeId,
						url,
						mimeType: response.mimeType || 'image/jpeg',
						fetchedAt: new Date(Number(response.fetchedAt))
					});

					prev.connected = true;
					return prev;
				});
			}
		}
	} catch (err) {
		if (err instanceof ConnectError) {
			if (err.code !== Code.Unknown && err.code !== Code.Canceled) {
				console.error('streamImages: error (' + err.code + ') ' + err.message);
			}
		}
	}

	update((prev) => {
		prev.connected = false;
		return prev;
	});

	return didConnect;
};

const backoffHandler = (backoff: number) => {
	update((prev) => {
		prev.backoff = backoff;
		return prev;
	});
};
