import { type Subscriber, writable } from 'svelte/store';
import { ConnectError, Code, type Client, type CallOptions } from '@connectrpc/connect';
import { create } from '@bufbuild/protobuf';
import { ClientsStreamRequestSchema, UserService } from '$lib/api/v1/clients/user_service_pb';
import { type Client as ClientModel } from '$lib/api/v1/clients/client_pb';
import { UserServiceClient } from './user-service-client';
import { Streamer, type HeartbeatHandler } from './streamer';
import { getAccessToken } from '$lib/stores/auth-store';

export type ClientsStoreType = {
	clientsConnected: boolean;
	clientsBackoff: number;
	clients: ClientModel[];
};

let clientsStreamer: Streamer<typeof UserService> | undefined = undefined;

const { subscribe, set, update } = writable<ClientsStoreType>(
	{
		clientsConnected: false,
		clientsBackoff: 0,
		clients: []
	},
	(set: Subscriber<ClientsStoreType>) => {
		if (clientsStreamer === undefined) {
			clientsStreamer = new Streamer('clients', UserServiceClient, streamClients, clientsBackoffHandler);
		} else {
			clientsStreamer.restart();
		}

		return () => {
			if (clientsStreamer !== undefined) {
				clientsStreamer.stop();
			}
		};
	}
);

export const ClientsStore = {
	subscribe
};

const streamClients = async (
	client: Client<typeof UserService>,
	abortSignal: AbortSignal,
	heartbeat: HeartbeatHandler
) => {
	let didConnect = false;
	const request = create(ClientsStreamRequestSchema, {});
	try {
		const options: CallOptions = {
			signal: abortSignal,
			headers: { authorization: getAccessToken() }
		};
		for await (const response of client.clientsStream(request, options)) {
			heartbeat();
			didConnect = true;

			if (response.id !== '') {
				update((prev: ClientsStoreType) => {
					const isRemoval =
						response.name === '' &&
						response.description === '' &&
						response.paired === false &&
						response.blocked === false &&
						response.online === false &&
						response.firstSeen === 0n &&
						response.lastSeen === 0n;

					if (isRemoval) {
						prev.clients = prev.clients.filter((item) => item.id !== response.id);
					} else {
						let found = false;
						for (let i = 0; i < prev.clients.length; i++) {
							if (prev.clients[i].id === response.id) {
								prev.clients[i] = response;
								found = true;
								break;
							}
						}
						if (!found) {
							prev.clients = [...prev.clients, response];
						}
					}

					prev.clients = prev.clients.sort((a, b) => {
						const aName = a.name ? a.name : a.id;
						const bName = b.name ? b.name : b.id;
						return aName > bName ? 1 : bName > aName ? -1 : 0;
					});

					prev.clientsConnected = true;
					return prev;
				});
			} else {
				update((prev: ClientsStoreType) => {
					prev.clientsConnected = true;
					return prev;
				});
			}
		}
	} catch (err) {
		if (err instanceof ConnectError) {
			if (err.code !== Code.Unknown && err.code !== Code.Canceled) {
				console.error('streamClients: error stream: (' + err.code + ') ' + err.message);
			}
		}
	}

	update((prev: ClientsStoreType) => {
		prev.clientsConnected = false;
		return prev;
	});

	return didConnect;
};

const clientsBackoffHandler = (backoff: number) => {
	update((prev: ClientsStoreType) => {
		prev.clientsBackoff = backoff;
		return prev;
	});
};
