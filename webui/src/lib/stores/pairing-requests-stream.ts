import { type Subscriber, writable } from 'svelte/store';
import { ConnectError, Code, type Client, type CallOptions } from '@connectrpc/connect';
import { create, toJsonString } from '@bufbuild/protobuf';
import {
	PairingRequestsStreamRequestSchema,
	PairingRequestsStreamResponseSchema,
	UserService,
	type PairingRequestsStreamResponse
} from '$lib/api/v1/clients/user_service_pb';
import type { PairingRequest } from '$lib/api/v1/clients/client_pb';
import { UserServiceClient } from './user-service-client';
import { Streamer, type HeartbeatHandler } from './streamer';
import { getAccessToken } from '$lib/stores/auth-store';

export type PairingRequestsStoreType = {
	connected: boolean;
	backoff: number;
	requests: PairingRequest[];
};

// We'll use a singleton streamer which will manage reconnections.
let streamer: Streamer<typeof UserService> | undefined = undefined;

// Create a writable store.
const { subscribe, update } = writable<PairingRequestsStoreType>(
	{ connected: false, backoff: 0, requests: [] },
	(set: Subscriber<PairingRequestsStoreType>) => {
		// console.log('pairing requests stream subscriber started');

		if (streamer === undefined) {
			streamer = new Streamer('pairing-requests', UserServiceClient, streamPairingRequests, backoffHandler);
		} else {
			streamer.restart();
		}

		return () => {
			// console.log('pairing requests stream subscriber finished');
			if (streamer !== undefined) {
				streamer.stop();
			}
		};
	}
);

export const PairingRequestsStore = {
	subscribe
};

const updateRequest = (prev: PairingRequest, next: PairingRequest): PairingRequest => {
	// No merging needed right now.
	return next;
};

const sortRequests = (requests: PairingRequest[]) => {
	return requests.sort((a, b) => {
		const aName = a.name || a.clientId;
		const bName = b.name || b.clientId;
		return aName > bName ? 1 : bName > aName ? -1 : 0;
	});
};

const streamPairingRequests = async (
	client: Client<typeof UserService>,
	abortSignal: AbortSignal,
	heartbeat: HeartbeatHandler
) => {
	let didConnect = false;
	const request = create(PairingRequestsStreamRequestSchema, {});
	try {
		const options: CallOptions = {
			signal: abortSignal,
			headers: { authorization: getAccessToken() }
		};
		let gotInitialSet = false; // Indicates that we've received the initial set of pairing requests from the server.
		let retainIDs: string[] = []; // Lists the request IDs that we should retain from the previous connection.

		for await (const response of client.pairingRequestsStream(request, options)) {
			heartbeat();
			didConnect = true;

			update((prev: PairingRequestsStoreType) => {
				console.log('pairing requests stream response: ', toJsonString(PairingRequestsStreamResponseSchema, response));

				if (response.pairingRequest !== undefined) {
					let found = false;
					for (let i = 0; i < prev.requests.length; i++) {
						if (prev.requests[i].requestId === response.pairingRequest?.requestId) {
							found = true;
							prev.requests[i] = updateRequest(prev.requests[i], response.pairingRequest);
							break;
						}
					}
					if (!found) {
						prev.requests = [...prev.requests, response.pairingRequest];
					}
					prev.requests = sortRequests(prev.requests);
					prev.connected = true;

					// If we're still receiving the initial set then store this
					// ID for device retention later.
					if (!gotInitialSet) {
						console.log('pairing requests stream: retaining request ID: ', response.pairingRequest.requestId);
						retainIDs = [...retainIDs, response.pairingRequest.requestId];
					}

					return prev;
				}

				if (response.pairingRemoved !== '') {
					for (let i = 0; i < prev.requests.length; i++) {
						if (prev.requests[i].requestId === response.pairingRemoved) {
							prev.requests.splice(i, 1);
							break;
						}
					}
					prev.connected = true;
					return prev;
				}

				// An empty message indicates the end of the initial batch of
				// requests after we connect (as well as heartbeats). Use this to
				// tidy up requests that were removed while we were not connected.
				if (!gotInitialSet) {
					gotInitialSet = true;
					console.log('pairing requests stream: initial set complete, retaining IDs: ', retainIDs);
					prev.requests = prev.requests.filter((request) => retainIDs.includes(request.requestId));
					retainIDs = []; // No longer needed.
				}

				// Keepalive
				prev.connected = true;
				return prev;
			});
		}
	} catch (err) {
		if (err instanceof ConnectError) {
			if (err.code !== Code.Unknown && err.code !== Code.Canceled) {
				console.error('streamPairingRequests: error stream: (' + err.code + ') ' + err.message);
			}
		}
	}

	update((prev: PairingRequestsStoreType) => {
		prev.connected = false;
		return prev;
	});

	return didConnect;
};

const backoffHandler = (backoff: number) => {
	update((prev: PairingRequestsStoreType) => {
		prev.backoff = backoff;
		return prev;
	});
};
