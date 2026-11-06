import { type Subscriber, writable } from 'svelte/store';
import { ConnectError, Code, type Client, type CallOptions } from '@connectrpc/connect';
import { create } from '@bufbuild/protobuf';
import { ImagesStreamRequestSchema, ImageSizeHintSchema, UserService } from '$lib/api/v2/clients/user_service_pb';
import { UserServiceClient } from './user-service-client';
import { Streamer, type HeartbeatHandler } from './streamer';

export type CachedImage = {
	deviceId: string;
	endpointId: string;
	serviceId: string;
	attributeId: string;
	/** Object URL created from image data — revoke when updating. */
	url: string;
	mimeType: string;
	fetchedAt: Date;
};

export type ImagesStoreType = {
	connected: boolean;
	backoff: number;
	/** Keyed by imageKey(deviceId, endpointId, serviceId, attributeId). */
	images: Map<string, CachedImage>;
};

/** The key of an image in ImagesStoreType.images. */
export function imageKey(deviceId: string, endpointId: string, serviceId: string, attributeId: string): string {
	return JSON.stringify([deviceId, endpointId, serviceId, attributeId]);
}

// ---------------------------------------------------------------------------
// Size hint registry
// Each camera component registers its rendered pixel dimensions here.
// key = imageKey(...)
// value = the image's IDs and a map of component-instance-id to {w, h}
// ---------------------------------------------------------------------------
type Dimensions = { w: number; h: number };
type HintEntry = {
	deviceId: string;
	endpointId: string;
	serviceId: string;
	attributeId: string;
	instances: Map<symbol, Dimensions>;
};
const hintRegistry = new Map<string, HintEntry>();

/** Returns the max width and height seen across all instances of an entry. */
function maxHintFor(entry: HintEntry): Dimensions {
	let maxW = 0,
		maxH = 0;
	for (const { w, h } of entry.instances.values()) {
		if (w > maxW) maxW = w;
		if (h > maxH) maxH = h;
	}
	return { w: maxW, h: maxH };
}

/** Collect all current hints as a flat array for the proto request. */
function currentHints() {
	const result: {
		deviceId: string;
		endpointId: string;
		serviceId: string;
		attributeId: string;
		w: number;
		h: number;
	}[] = [];
	for (const entry of hintRegistry.values()) {
		const { w, h } = maxHintFor(entry);
		if (w > 0 || h > 0) {
			result.push({
				deviceId: entry.deviceId,
				endpointId: entry.endpointId,
				serviceId: entry.serviceId,
				attributeId: entry.attributeId,
				w,
				h
			});
		}
	}
	return result;
}

/**
 * Register a component instance's rendered size for a camera key.
 * Call unregisterSizeHint on component destroy.
 */
export function registerSizeHint(
	deviceId: string,
	endpointId: string,
	serviceId: string,
	attributeId: string,
	instanceId: symbol,
	w: number,
	h: number
): void {
	const key = imageKey(deviceId, endpointId, serviceId, attributeId);
	let entry = hintRegistry.get(key);
	if (!entry) {
		entry = { deviceId, endpointId, serviceId, attributeId, instances: new Map() };
		hintRegistry.set(key, entry);
	}
	entry.instances.set(instanceId, { w, h });
	// Restart stream so the server learns the new max size.
	restartStream();
}

export function unregisterSizeHint(
	deviceId: string,
	endpointId: string,
	serviceId: string,
	attributeId: string,
	instanceId: symbol
): void {
	const key = imageKey(deviceId, endpointId, serviceId, attributeId);
	const entry = hintRegistry.get(key);
	entry?.instances.delete(instanceId);
	if (entry?.instances.size === 0) hintRegistry.delete(key);
	restartStream();
}

// ---------------------------------------------------------------------------
// Streamer + writable store
// ---------------------------------------------------------------------------
let streamer: Streamer<typeof UserService> | undefined = undefined;

let reconnectTimer: ReturnType<typeof setTimeout> | undefined;

function restartStream() {
	if (streamer === undefined) return;
	// Debounce: ResizeObserver fires rapidly during layout; wait for it to settle.
	if (reconnectTimer !== undefined) clearTimeout(reconnectTimer);
	reconnectTimer = setTimeout(() => {
		reconnectTimer = undefined;
		streamer?.reconnect();
	}, 200);
}

const { subscribe, update } = writable<ImagesStoreType>(
	{ connected: false, backoff: 0, images: new Map() },
	(_set: Subscriber<ImagesStoreType>) => {
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

// ---------------------------------------------------------------------------
// Streaming function — called fresh on every connect/reconnect
// ---------------------------------------------------------------------------
const streamImages = async (
	client: Client<typeof UserService>,
	abortSignal: AbortSignal,
	heartbeat: HeartbeatHandler
) => {
	let didConnect = false;

	// Snapshot current hints at connection time.
	const hints = currentHints().map(({ deviceId, endpointId, serviceId, attributeId, w, h }) =>
		create(ImageSizeHintSchema, {
			deviceId,
			endpointId,
			serviceId,
			attributeId,
			width: w,
			height: h
		})
	);

	const request = create(ImagesStreamRequestSchema, { sizeHints: hints });

	try {
		const options: CallOptions = {
			signal: abortSignal
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

			const key = imageKey(response.deviceId, response.endpointId, response.serviceId, response.attributeId);

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
						endpointId: response.endpointId,
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
			if (err.code !== Code.Unknown && err.code !== Code.Canceled && err.code !== Code.Unauthenticated) {
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
