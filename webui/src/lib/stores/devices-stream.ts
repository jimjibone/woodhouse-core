import { type Subscriber, writable } from 'svelte/store';
import { ConnectError, Code, type Client, type CallOptions } from '@connectrpc/connect';
import { create } from '@bufbuild/protobuf';
import { DevicesStreamRequestSchema, UserService } from '$lib/api/v2/clients/user_service_pb';
import {
	EventCursorSchema,
	type Device,
	type Device_DeviceType,
	type Endpoint,
	type Event,
	type EventCursor,
	type Service,
	type TimeValue
} from '$lib/api/v2/clients/client_service_pb';
import { Info, Online, Battery } from '$lib/schema/zz_services';
import { ROOT_ENDPOINT_ID, asBool, asInt, asText, asTimeValue, findAttr, iterServices } from '$lib/device';
import { UserServiceClient } from './user-service-client';
import { Streamer, type HeartbeatHandler } from './streamer';

export type DevicesStoreType = {
	connected: boolean;
	backoff: number;
	devices: DevicesStoreDevice[];
	/**
	 * Set when the server said some events since our cursor are gone (it
	 * restarted, or its buffer no longer reaches back that far). Cleared on
	 * the next connection. The events we already hold are kept.
	 */
	eventsLost: boolean;
};

export type DevicesStoreDevice = {
	id: string;
	clientID: string;
	typ: Device_DeviceType;
	/** The device name from the info service, the ID until it is known. */
	name: string | undefined;
	online: boolean;
	batteryLevel: bigint | undefined;
	lastSeen: TimeValue | undefined;
	endpoints: Endpoint[];
	/** Recent events, newest first, at most MAX_EVENTS. Never part of device state. */
	events: Event[];
};

/** How many events are kept per device. */
export const MAX_EVENTS = 50;

// We'll use a singleton streamer which will manage reconnections.
let streamer: Streamer<typeof UserService> | undefined = undefined;

// Create a writable store.
const { subscribe, update } = writable<DevicesStoreType>(
	{ connected: false, backoff: 0, devices: [], eventsLost: false },
	(_set: Subscriber<DevicesStoreType>) => {
		console.log('devices stream subscriber started');

		if (streamer === undefined) {
			streamer = new Streamer('devices', UserServiceClient, streamDevices, backoffHandler);
		} else {
			streamer.restart();
		}

		return () => {
			console.log('devices stream subscriber finished');
			if (streamer !== undefined) {
				streamer.stop();
			}
		};
	}
);

export const DevicesStore = {
	subscribe
};

//****************************** Live events ******************************//

export type LiveEventHandler = (deviceID: string, event: Event) => void;

const liveEventHandlers = new Set<LiveEventHandler>();

/**
 * Subscribes to live events: those that happened while connected, not the
 * ones replayed from the server's buffer (which have `replayed` set). Returns
 * the unsubscribe function. The handler is not called for events that
 * arrived before it subscribed.
 */
export const subscribeLiveEvents = (handler: LiveEventHandler): (() => void) => {
	liveEventHandlers.add(handler);
	return () => {
		liveEventHandlers.delete(handler);
	};
};

//****************************** Merging ******************************//

// Where we got up to in the server's event history. Sent as events_since when
// the stream reconnects. Undefined until the first event is seen, in which
// case the whole buffer is requested.
let lastCursor: { epoch: string; seq: bigint } | undefined = undefined;

const noteCursor = (ev: Event) => {
	if (ev.epoch === '' && ev.seq === 0n) return;
	if (lastCursor === undefined || lastCursor.epoch !== ev.epoch || ev.seq > lastCursor.seq) {
		lastCursor = { epoch: ev.epoch, seq: ev.seq };
	}
};

const eventCursor = (): EventCursor =>
	create(EventCursorSchema, lastCursor ?? { epoch: '', seq: 0n });

/**
 * Adds events to a newest-first list, skipping ones already held (same epoch
 * and seq) and capping the length. Events of one epoch are ordered by seq;
 * events of a newer epoch (the server restarted) go in front.
 */
export const mergeEvents = (held: Event[], incoming: Event[]): Event[] => {
	let out = held;
	for (const ev of incoming) {
		if (out.some((e) => e.epoch === ev.epoch && e.seq === ev.seq)) continue;
		let at = out.findIndex((e) => e.epoch !== ev.epoch || e.seq < ev.seq);
		if (at < 0) at = out.length;
		out = [...out.slice(0, at), ev, ...out.slice(at)];
	}
	return out.length > MAX_EVENTS ? out.slice(0, MAX_EVENTS) : out;
};

// Events can arrive for a device before its first snapshot. They wait here
// instead of creating a device that has no state yet.
const pendingEvents = new Map<string, Event[]>();

/**
 * Merges the endpoints of a device update into the held ones.
 *
 * A full_state update replaces all endpoints. A partial update carries only
 * the changed endpoints and each listed service whole, so listed services
 * replace the held ones with the same ID. An endpoint with no services is a
 * label update: the labels are merged and the services kept.
 */
export const mergeEndpoints = (held: Endpoint[], next: Device): Endpoint[] => {
	if (next.fullState) {
		return [...next.endpoints];
	}
	if (next.endpoints.length === 0) {
		return held;
	}

	const out = [...held];
	for (const ne of next.endpoints) {
		const at = out.findIndex((ep) => ep.id === ne.id);
		if (at < 0) {
			out.push(ne);
			continue;
		}
		const pe = out[at];
		if (ne.services.length === 0) {
			// Label update: the labels are authoritative (empty clears).
			out[at] = { ...pe, label: ne.label, userLabel: ne.userLabel };
			continue;
		}
		const services: Service[] = [...pe.services];
		for (const ns of ne.services) {
			const si = services.findIndex((s) => s.id === ns.id);
			if (si < 0) {
				services.push(ns);
			} else {
				services[si] = ns;
			}
		}
		out[at] = {
			...pe,
			label: ne.label || pe.label,
			userLabel: ne.userLabel || pe.userLabel,
			services
		};
	}
	return out;
};

type Summary = Pick<DevicesStoreDevice, 'name' | 'online' | 'batteryLevel' | 'lastSeen'>;

const newer = (a: TimeValue | undefined, b: TimeValue | undefined): TimeValue | undefined => {
	if (a === undefined) return b;
	if (b === undefined) return a;
	return b.seconds > a.seconds || (b.seconds === a.seconds && b.nanos > a.nanos) ? b : a;
};

// The device-level summary comes from the info, online and battery services.
// The root endpoint is looked at first, so a device with a battery per
// endpoint reports the root's. Last seen is the newest of online.last_seen and
// the observation time of any other attribute; the online service's own
// attribute times are skipped, as the server stamps them when it marks the
// device offline.
const summarise = (id: string, endpoints: Endpoint[]): Summary => {
	const sum: Summary = { name: id, online: false, batteryLevel: undefined, lastSeen: undefined };
	let gotName = false;
	let gotOnline = false;
	let gotBattery = false;
	for (const { endpoint, service } of iterServices({ endpoints })) {
		if (service.typ !== Online.type) {
			for (const attr of service.attrs) {
				sum.lastSeen = newer(sum.lastSeen, attr.time);
			}
		}
		if (service.typ === Info.type && !gotName) {
			const name = asText(findAttr(service, Info.attributes.name)?.value);
			if (name !== undefined) {
				sum.name = name;
				gotName = true;
			}
		} else if (service.typ === Online.type && !gotOnline && endpoint.id === ROOT_ENDPOINT_ID) {
			sum.online = asBool(findAttr(service, Online.attributes.online)?.value) ?? false;
			sum.lastSeen = newer(sum.lastSeen, asTimeValue(findAttr(service, Online.attributes.lastSeen)?.value));
			gotOnline = true;
		} else if (service.typ === Battery.type && !gotBattery) {
			const level = asInt(findAttr(service, Battery.attributes.level)?.value);
			if (level !== undefined) {
				sum.batteryLevel = level;
				gotBattery = true;
			}
		}
	}
	return sum;
};

/**
 * Applies a device update to the held device (undefined if none) and returns
 * the new device. The held device is not modified.
 */
export const mergeDevice = (held: DevicesStoreDevice | undefined, next: Device): DevicesStoreDevice => {
	const endpoints = mergeEndpoints(held?.endpoints ?? [], next);
	const events = mergeEvents(held?.events ?? [], next.events);
	return {
		id: next.id,
		clientID: next.clientId || held?.clientID || '',
		typ: next.typ,
		...summarise(next.id, endpoints),
		endpoints,
		events
	};
};

const byName = (a: DevicesStoreDevice, b: DevicesStoreDevice) => {
	const aName = a.name ? a.name : a.id;
	const bName = b.name ? b.name : b.id;
	return aName > bName ? 1 : bName > aName ? -1 : 0;
};

//****************************** Stream ******************************//

const streamDevices = async (
	client: Client<typeof UserService>,
	abortSignal: AbortSignal,
	heartbeat: HeartbeatHandler
) => {
	let didConnect = false;

	// Ask for the whole event buffer the first time, and everything since the
	// last event we saw when reconnecting.
	const request = create(DevicesStreamRequestSchema, { eventsSince: eventCursor() });
	update((prev) => ({ ...prev, eventsLost: false }));

	try {
		const options: CallOptions = {
			signal: abortSignal
		};
		let gotInitialSet = false;
		let retainIDs: string[] = [];

		for await (const response of client.devicesStream(request, options)) {
			heartbeat();
			didConnect = true;

			if (response.eventsLost) {
				update((prev) => ({ ...prev, eventsLost: true }));
			}

			if (response.device !== undefined) {
				const next = response.device;
				const fresh: Event[] = [];
				update((prev: DevicesStoreType) => {
					const at = prev.devices.findIndex((d) => d.id === next.id);
					const held = at >= 0 ? prev.devices[at] : undefined;

					// Events for a device we have no state for yet wait for its
					// snapshot. Hold them back from the merge until then.
					let toMerge = next;
					if (held === undefined && next.endpoints.length === 0 && !next.fullState) {
						if (next.events.length > 0) {
							pendingEvents.set(next.id, mergeEvents(pendingEvents.get(next.id) ?? [], next.events));
						}
						for (const ev of next.events) {
							noteCursor(ev);
							fresh.push(ev);
						}
						return { ...prev, connected: true };
					}
					if (held === undefined) {
						const pending = pendingEvents.get(next.id);
						if (pending !== undefined) {
							pendingEvents.delete(next.id);
							toMerge = { ...next, events: [...pending].reverse().concat(next.events) };
						}
					}

					// Track what is genuinely new, so live handlers only see
					// events once and never ones already held.
					for (const ev of next.events) {
						noteCursor(ev);
						if (!held?.events.some((e) => e.epoch === ev.epoch && e.seq === ev.seq)) {
							fresh.push(ev);
						}
					}

					const merged = mergeDevice(held, toMerge);
					const devices = [...prev.devices];
					if (at >= 0) {
						devices[at] = merged;
					} else {
						devices.push(merged);
					}
					devices.sort(byName);

					// If we're still receiving the initial set then store this
					// ID for device retention later.
					if (!gotInitialSet) {
						retainIDs = [...retainIDs, next.id];
					}

					return { ...prev, devices, connected: true };
				});

				for (const ev of fresh) {
					if (ev.replayed) continue;
					for (const handler of [...liveEventHandlers]) {
						try {
							handler(next.id, ev);
						} catch (err) {
							console.error('streamDevices: live event handler failed:', err);
						}
					}
				}
			} else if (response.deviceRemoved !== '') {
				pendingEvents.delete(response.deviceRemoved);
				update((prev: DevicesStoreType) => ({
					...prev,
					devices: prev.devices.filter((d) => d.id !== response.deviceRemoved),
					connected: true
				}));
			} else if (!response.eventsLost) {
				// An empty message indicates the end of the initial batch of
				// devices after we connect (as well as heartbeats). Use this to
				// tidy up devices that were removed while we were not listening.
				if (!gotInitialSet) {
					gotInitialSet = true;

					const retained = new Set(retainIDs);
					retainIDs = [];
					for (const id of [...pendingEvents.keys()]) {
						if (!retained.has(id)) pendingEvents.delete(id);
					}
					update((prev: DevicesStoreType) => ({
						...prev,
						devices: prev.devices.filter((d) => retained.has(d.id)),
						connected: true
					}));
				} else {
					update((prev: DevicesStoreType) => ({ ...prev, connected: true }));
				}
			}
		}
	} catch (err) {
		if (err instanceof ConnectError) {
			if (err.code !== Code.Unknown && err.code !== Code.Canceled && err.code !== Code.Unauthenticated) {
				console.error('streamDevices: error stream: (' + err.code + ') ' + err.message);
			}
		}
	}

	update((prev: DevicesStoreType) => ({ ...prev, connected: false }));

	return didConnect;
};

const backoffHandler = (backoff: number) => {
	update((prev: DevicesStoreType) => ({ ...prev, backoff }));
};
