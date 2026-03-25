import { type Subscriber, writable } from 'svelte/store';
import { ConnectError, Code, type Client, type CallOptions } from '@connectrpc/connect';
import { create } from '@bufbuild/protobuf';
import { GroupsStreamRequestSchema, UserService } from '$lib/api/v1/clients/user_service_pb';
import type { Group } from '$lib/api/v1/clients/group_pb';
import { UserServiceClient } from './user-service-client';
import { Streamer, type HeartbeatHandler } from './streamer';
import { getAccessToken } from '$lib/stores/auth-store';

export type GroupsStoreType = {
	connected: boolean;
	backoff: number;
	groups: Group[];
	error: string | null;
};

let streamer: Streamer<typeof UserService> | undefined = undefined;

const { subscribe, set, update } = writable<GroupsStoreType>(
	{ connected: false, backoff: 0, groups: [], error: null },
	(set: Subscriber<GroupsStoreType>) => {
		if (streamer === undefined) {
			streamer = new Streamer('groups', UserServiceClient, streamGroups, backoffHandler);
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

export const GroupsStore = {
	subscribe
};

const streamGroups = async (
	client: Client<typeof UserService>,
	abortSignal: AbortSignal,
	heartbeat: HeartbeatHandler
) => {
	let didConnect = false;

	const request = create(GroupsStreamRequestSchema, {});
	try {
		const options: CallOptions = {
			signal: abortSignal,
			headers: { authorization: getAccessToken() }
		};
		let gotInitialSet = false;
		let retainIDs: string[] = [];

		for await (const response of client.groupsStream(request, options)) {
			heartbeat();
			didConnect = true;

			if (response.groupUpdate !== undefined) {
				update((prev: GroupsStoreType) => {
					let found = false;
					for (let d = 0; d < prev.groups.length; d++) {
						if (prev.groups[d].id === response.groupUpdate?.id) {
							found = true;
							prev.groups[d] = response.groupUpdate!;
							break;
						}
					}
					if (!found) {
						prev.groups = [...prev.groups, response.groupUpdate!];
					}

					// If we're still receiving the initial set then store this
					// ID for group retention later.
					if (!gotInitialSet) {
						retainIDs = [...retainIDs, response.groupUpdate!.id];
					}

					prev.groups = prev.groups.sort((a, b) => {
						return a.name > b.name ? 1 : b.name > a.name ? -1 : 0;
					});

					prev.connected = true;
					prev.error = null;
					return prev;
				});
			} else if (response.removedId !== '') {
				update((prev: GroupsStoreType) => {
					for (let d = 0; d < prev.groups.length; d++) {
						if (prev.groups[d].id === response.removedId) {
							prev.groups.splice(d, 1);
							break;
						}
					}
					prev.connected = true;
					prev.error = null;
					return prev;
				});
			} else {
				update((prev: GroupsStoreType) => {
					prev.connected = true;
					prev.error = null;
					return prev;
				});

				// An empty message indicates the end of the initial batch of
				// groups after we connect (as well as heartbeats). Use this to
				// tidy up groups that were removed while we were not listening.
				if (!gotInitialSet) {
					gotInitialSet = true;

					// Remove any groups not in the retain list.
					update((prev: GroupsStoreType) => {
						for (let d = 0; d < prev.groups.length; d++) {
							if (!retainIDs.includes(prev.groups[d].id)) {
								// console.log('streamGroups: not retained ' + prev.groups[d].id);
								prev.groups.splice(d, 1);
								d--;
							}
						}

						// Don't need this anymore.
						retainIDs = [];

						prev.connected = true;
						prev.error = null;
						return prev;
					});
				} else {
					update((prev: GroupsStoreType) => {
						prev.connected = true;
						prev.error = null;
						return prev;
					});
				}
			}
		}
	} catch (err) {
		if (err instanceof ConnectError) {
			if (err.code !== Code.Unknown && err.code !== Code.Canceled) {
				console.error('streamGroups: error stream: (' + err.code + ') ' + err.message);
				const msg = err.rawMessage || err.message;
				update((prev: GroupsStoreType) => {
					prev.error = msg;
					return prev;
				});
			}
		}
	}

	update((prev: GroupsStoreType) => {
		prev.connected = false;
		return prev;
	});

	return didConnect;
};

const backoffHandler = (backoff: number) => {
	update((prev: GroupsStoreType) => {
		prev.backoff = backoff;
		return prev;
	});
};
