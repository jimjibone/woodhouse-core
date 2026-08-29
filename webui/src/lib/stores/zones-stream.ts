import { type Subscriber, writable } from 'svelte/store';
import { ConnectError, Code, type Client, type CallOptions } from '@connectrpc/connect';
import { create } from '@bufbuild/protobuf';
import { ZonesStreamRequestSchema, UserService } from '$lib/api/v1/clients/user_service_pb';
import type { Zone } from '$lib/api/v1/clients/zone_pb';
import { UserServiceClient } from './user-service-client';
import { Streamer, type HeartbeatHandler } from './streamer';

export type ZonesStoreType = {
	connected: boolean;
	backoff: number;
	zones: Zone[];
	error: string | null;
};

let streamer: Streamer<typeof UserService> | undefined = undefined;

const { subscribe, update } = writable<ZonesStoreType>(
	{ connected: false, backoff: 0, zones: [], error: null },
	// eslint-disable-next-line @typescript-eslint/no-unused-vars
	(set: Subscriber<ZonesStoreType>) => {
		if (streamer === undefined) {
			streamer = new Streamer('zones', UserServiceClient, streamZones, backoffHandler);
		} else {
			streamer.restart();
		}

		return () => {
			if (streamer !== undefined) {
				streamer.stop();
			}
		};
	}
);

export const ZonesStore = {
	subscribe
};

const byName = (a: Zone, b: Zone) => (a.name > b.name ? 1 : b.name > a.name ? -1 : 0);

const markConnected = (prev: ZonesStoreType): ZonesStoreType => {
	prev.connected = true;
	prev.error = null;
	return prev;
};

const streamZones = async (
	client: Client<typeof UserService>,
	abortSignal: AbortSignal,
	heartbeat: HeartbeatHandler
) => {
	let didConnect = false;

	const request = create(ZonesStreamRequestSchema, {});
	try {
		const options: CallOptions = {
			signal: abortSignal
		};
		let gotInitialSet = false;
		let retainIDs: string[] = [];

		for await (const response of client.zonesStream(request, options)) {
			// Every message is proof the stream is alive, not just the heartbeat.
			heartbeat();
			didConnect = true;

			// ZonesStreamResponse is a oneof, so exactly one of these is set.
			// No need to tell an update from a removal from a keepalive by
			// checking for empty values, the way the older streams do.
			switch (response.update.case) {
				case 'zoneUpdate': {
					const zone = response.update.value;
					update((prev: ZonesStoreType) => {
						const at = prev.zones.findIndex((z) => z.id === zone.id);
						if (at >= 0) {
							prev.zones[at] = zone;
						} else {
							prev.zones = [...prev.zones, zone];
						}

						// While the initial set is still arriving, note the ID so
						// we can drop zones deleted while we were not listening.
						if (!gotInitialSet) {
							retainIDs = [...retainIDs, zone.id];
						}

						prev.zones = prev.zones.sort(byName);
						return markConnected(prev);
					});
					break;
				}

				case 'removedId': {
					const removedID = response.update.value;
					update((prev: ZonesStoreType) => {
						prev.zones = prev.zones.filter((z) => z.id !== removedID);
						return markConnected(prev);
					});
					break;
				}

				case 'heartbeat': {
					update(markConnected);

					// The first heartbeat marks the end of the initial batch, so
					// anything we still hold that the server did not send is
					// gone. Drop it.
					if (!gotInitialSet) {
						gotInitialSet = true;
						update((prev: ZonesStoreType) => {
							prev.zones = prev.zones.filter((z) => retainIDs.includes(z.id));
							retainIDs = [];
							return markConnected(prev);
						});
					}
					break;
				}
			}
		}
	} catch (err) {
		if (err instanceof ConnectError) {
			if (err.code !== Code.Unknown && err.code !== Code.Canceled && err.code !== Code.Unauthenticated) {
				console.error('streamZones: error stream: (' + err.code + ') ' + err.message);
				const msg = err.rawMessage || err.message;
				update((prev: ZonesStoreType) => {
					prev.error = msg;
					return prev;
				});
			}
		}
	}

	update((prev: ZonesStoreType) => {
		prev.connected = false;
		return prev;
	});

	return didConnect;
};

const backoffHandler = (backoff: number) => {
	update((prev: ZonesStoreType) => {
		prev.backoff = backoff;
		return prev;
	});
};
