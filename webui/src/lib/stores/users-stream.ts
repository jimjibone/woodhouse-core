import { type Subscriber, writable } from 'svelte/store';
import { ConnectError, Code, type Client, type CallOptions } from '@connectrpc/connect';
import { create, toJsonString } from '@bufbuild/protobuf';
import { UsersStreamResponseSchema, UserService, type User } from '$lib/api/v1/clients/user_service_pb';
import { type DeviceService, UsersStreamRequestSchema } from '$lib/api/v1/clients/user_service_pb';
import { UserServiceClient } from './user-service-client';
import { Streamer, type HeartbeatHandler } from './streamer';
import { getAccessToken } from '$lib/stores/auth-store';

export type UsersStoreType = {
	connected: boolean;
	backoff: number;
	users: User[];
};

// We'll use a singleton streamer which will manage reconnections.
// let streamer: Streamer | undefined = undefined;
let streamer: Streamer<typeof UserService> | undefined = undefined;

// Create a writable store.
const { subscribe, set, update } = writable<UsersStoreType>(
	{ connected: false, backoff: 0, users: [] },
	(set: Subscriber<UsersStoreType>) => {
		// console.log('users stream subscriber started');

		if (streamer === undefined) {
			streamer = new Streamer('faves', UserServiceClient, streamUsers, backoffHandler);
		} else {
			streamer.restart();
		}

		return () => {
			// console.log('users stream subscriber finished');
			if (streamer !== undefined) {
				streamer.stop();
			}
		};
	}
);

export const UsersStore = {
	subscribe
};

const updateUser = (prev: User, next: User): User => {
	// No merging required at the moment...
	return next;
};

const streamUsers = async (
	client: Client<typeof UserService>,
	abortSignal: AbortSignal,
	heartbeat: HeartbeatHandler
) => {
	let didConnect = false;
	const request = create(UsersStreamRequestSchema, {});
	try {
		// console.log('streamUsers: starting stream');
		const options: CallOptions = {
			signal: abortSignal,
			headers: { authorization: getAccessToken() }
		};
		for await (const response of client.usersStream(request, options)) {
			heartbeat();
			didConnect = true;

			// All fields will be empty if this is a keepalive message.
			if (response.user !== undefined) {
				// console.log('streamUsers: update: ' + toJsonString(UsersStreamResponseSchema, response));
				update((prev: UsersStoreType) => {
					let foundUser = false;
					for (let d = 0; d < prev.users.length; d++) {
						if (prev.users[d].username === response.user?.username) {
							// console.log('streamUsers: update: updating "' + response.user?.username + '"');
							foundUser = true;
							prev.users[d] = updateUser(prev.users[d], response.user);
							break;
						}
					}
					if (!foundUser) {
						// console.log('streamUsers: update: new to us "' + response.user?.username + '"');
						prev.users = [...prev.users, response.user!];
					}

					prev.users = prev.users.sort((a, b) => {
						const aName = a.username;
						const bName = b.username;
						return aName > bName ? 1 : bName > aName ? -1 : 0;
					});

					prev.connected = true;

					return prev;
				});
			} else if (response.userRemoved !== '') {
				// console.log('streamUsers: removed "' + response.userRemoved + '"');
				update((prev: UsersStoreType) => {
					for (let d = 0; d < prev.users.length; d++) {
						if (prev.users[d].username === response.userRemoved) {
							// console.log('streamUsers: removed: found ' + response.userRemoved);
							prev.users.splice(d, 1);
							break;
						}
					}

					prev.connected = true;

					return prev;
				});
			} else {
				update((prev: UsersStoreType) => {
					prev.connected = true;
					return prev;
				});
			}
		}
	} catch (err) {
		if (err instanceof ConnectError) {
			if (err.code !== Code.Unknown && err.code !== Code.Canceled) {
				console.error('streamUsers: error stream: (' + err.code + ') ' + err.message);
			}
		}
	}

	update((prev: UsersStoreType) => {
		prev.connected = false;
		return prev;
	});

	return didConnect;
};

const backoffHandler = (backoff: number) => {
	update((prev: UsersStoreType) => {
		prev.backoff = backoff;
		return prev;
	});
};
