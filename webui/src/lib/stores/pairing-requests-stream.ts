import { type Subscriber, writable } from 'svelte/store';
import { ConnectError, Code, type Client, type CallOptions } from '@connectrpc/connect';
import { create } from '@bufbuild/protobuf';
import {
	PairingRequestsStreamRequestSchema,
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

const handleResponse = (response: PairingRequestsStreamResponse) => {
	update((prev: PairingRequestsStoreType) => {
		if (response.pairingRequest !== undefined) {
			let found = false;
			for (let i = 0; i < prev.requests.length; i++) {
				if (prev.requests[i].clientId === response.pairingRequest?.clientId) {
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
			return prev;
		}

		if (response.pairingRemoved !== '') {
			for (let i = 0; i < prev.requests.length; i++) {
				if (prev.requests[i].clientId === response.pairingRemoved) {
					prev.requests.splice(i, 1);
					break;
				}
			}
			prev.connected = true;
			return prev;
		}

		// Keepalive
		prev.connected = true;
		return prev;
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
		for await (const response of client.pairingRequestsStream(request, options)) {
			heartbeat();
			didConnect = true;
			handleResponse(response);
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
