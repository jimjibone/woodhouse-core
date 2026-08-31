import { type Subscriber, writable } from 'svelte/store';
import { ConnectError, Code, type Client, type CallOptions } from '@connectrpc/connect';
import { create } from '@bufbuild/protobuf';
import { NotificationsStreamRequestSchema, NotificationLevel, UserService } from '$lib/api/v1/clients/user_service_pb';
import type { Notification } from '$lib/api/v1/clients/user_service_pb';
import { UserServiceClient } from './user-service-client';
import { Streamer, type HeartbeatHandler } from './streamer';
import { toast } from 'svelte-sonner';

export type NotificationsStoreType = {
	connected: boolean;
	backoff: number;
	notifications: Notification[];
	unread: number;
	error: string | null;
};

let streamer: Streamer<typeof UserService> | undefined = undefined;

const { subscribe, update } = writable<NotificationsStoreType>(
	{ connected: false, backoff: 0, notifications: [], unread: 0, error: null },
	// eslint-disable-next-line @typescript-eslint/no-unused-vars
	(set: Subscriber<NotificationsStoreType>) => {
		if (streamer === undefined) {
			streamer = new Streamer('notifications', UserServiceClient, streamNotifications, backoffHandler);
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

export const NotificationsStore = {
	subscribe
};

// Newest first. createdAt is an int64, which protobuf-es maps to bigint, so it
// has to be widened before arithmetic.
const byNewest = (a: Notification, b: Notification) => Number(b.createdAt) - Number(a.createdAt);

export const notificationDate = (notification: Notification): Date => new Date(Number(notification.createdAt));

const markConnected = (prev: NotificationsStoreType): NotificationsStoreType => {
	prev.connected = true;
	prev.error = null;
	return prev;
};

const recount = (prev: NotificationsStoreType): NotificationsStoreType => {
	prev.unread = prev.notifications.filter((n) => !n.read).length;
	return prev;
};

const toastFor = (notification: Notification) => {
	const options = notification.body ? { description: notification.body } : undefined;
	switch (notification.level) {
		case NotificationLevel.SUCCESS:
			toast.success(notification.title, options);
			break;
		case NotificationLevel.WARNING:
			toast.warning(notification.title, options);
			break;
		case NotificationLevel.ERROR:
			toast.error(notification.title, options);
			break;
		default:
			toast.info(notification.title, options);
			break;
	}
};

const streamNotifications = async (
	client: Client<typeof UserService>,
	abortSignal: AbortSignal,
	heartbeat: HeartbeatHandler
) => {
	let didConnect = false;

	const request = create(NotificationsStreamRequestSchema, {});
	try {
		const options: CallOptions = {
			signal: abortSignal
		};
		// Local to this call, so it is false again on every reconnect - which is
		// exactly right, since each reconnect gets a fresh initial batch.
		let gotInitialSet = false;
		let retainIDs: string[] = [];

		for await (const response of client.notificationsStream(request, options)) {
			// Every message is proof the stream is alive, not just the heartbeat.
			heartbeat();
			didConnect = true;

			switch (response.update.case) {
				case 'notificationUpdate': {
					const notification = response.update.value;
					update((prev: NotificationsStoreType) => {
						const at = prev.notifications.findIndex((n) => n.id === notification.id);
						const isNew = at < 0;
						if (isNew) {
							prev.notifications = [notification, ...prev.notifications];
						} else {
							prev.notifications[at] = notification;
						}

						if (!gotInitialSet) {
							// Still replaying history. Note the ID so we can drop
							// anything deleted while we were not listening, and do
							// not toast - the user has seen these, or was not here
							// for them. The bell shows them as unread instead.
							retainIDs = [...retainIDs, notification.id];
						} else if (isNew && !notification.read) {
							// A read flip from another device arrives here as a
							// full update on a row we already hold. Without the
							// isNew check, marking something read on the phone
							// would pop a toast in the browser.
							toastFor(notification);
						}

						prev.notifications = prev.notifications.sort(byNewest);
						return recount(markConnected(prev));
					});
					break;
				}

				case 'removedId': {
					const removedID = response.update.value;
					update((prev: NotificationsStoreType) => {
						prev.notifications = prev.notifications.filter((n) => n.id !== removedID);
						return recount(markConnected(prev));
					});
					break;
				}

				case 'heartbeat': {
					update(markConnected);

					// The first heartbeat marks the end of the initial batch, so
					// anything we still hold that the server did not send is gone.
					// Drop it. Later heartbeats must not prune - by then the batch
					// is over and retainIDs is empty.
					if (!gotInitialSet) {
						gotInitialSet = true;
						update((prev: NotificationsStoreType) => {
							prev.notifications = prev.notifications.filter((n) => retainIDs.includes(n.id));
							retainIDs = [];
							return recount(markConnected(prev));
						});
					}
					break;
				}
			}
		}
	} catch (err) {
		if (err instanceof ConnectError) {
			if (err.code !== Code.Unknown && err.code !== Code.Canceled && err.code !== Code.Unauthenticated) {
				console.error('streamNotifications: error stream: (' + err.code + ') ' + err.message);
				const msg = err.rawMessage || err.message;
				update((prev: NotificationsStoreType) => {
					prev.error = msg;
					return prev;
				});
			}
		}
	}

	update((prev: NotificationsStoreType) => {
		prev.connected = false;
		return prev;
	});

	return didConnect;
};

const backoffHandler = (backoff: number) => {
	update((prev: NotificationsStoreType) => {
		prev.backoff = backoff;
		return prev;
	});
};
