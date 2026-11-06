import {
	ActionRequestSchema,
	ActionResponseSchema,
	ActionResponse_ActionStatus,
	CommandActionSchema,
	ImageResponseSchema,
	ImageResponse_ImageStatus,
	WriteActionSchema,
	type FieldValue,
	type ActionResponse,
	type ImageResponse,
	Service_ServiceType
} from '$lib/api/v2/clients/client_service_pb';
import {
	AddFavoriteRequestSchema,
	AddGroupRequestSchema,
	DismissNotificationRequestSchema,
	MarkAllNotificationsReadRequestSchema,
	MarkNotificationReadRequestSchema,
	SendTestNotificationRequestSchema,
	AddGroupResponseSchema,
	AddUserRequestSchema,
	AddUserResponseSchema,
	ApprovePairingRequestSchema,
	ApprovePairingResponseSchema,
	DenyPairingRequestSchema,
	DenyPairingResponseSchema,
	RemoveDeviceRequestSchema,
	RemoveFavoriteRequestSchema,
	RemoveGroupRequestSchema,
	RemoveGroupResponseSchema,
	UnpairClientRequestSchema,
	UnpairClientResponseSchema,
	ForgetClientRequestSchema,
	ForgetClientResponseSchema,
	UpdateGroupRequestSchema,
	UpdateGroupResponseSchema,
	AddZoneRequestSchema,
	AddZoneResponseSchema,
	UpdateZoneRequestSchema,
	UpdateZoneResponseSchema,
	RemoveZoneRequestSchema,
	RemoveZoneResponseSchema,
	UpdateUserRequestSchema,
	UpdateUserResponseSchema,
	UserImageRequestSchema,
	UserRole,
	NotificationLevel,
	NotificationAudience_Kind,
	GetSettingsRequestSchema,
	UpdateSettingsRequestSchema,
	SetEndpointLabelRequestSchema,
	type Notification,
	type UpdateUserRequest
} from '$lib/api/v2/clients/user_service_pb';
import { type GroupMember } from '$lib/api/v2/clients/group_pb';
import { type EndpointRef } from '$lib/api/v2/clients/zone_pb';
import { UserServiceClient } from './user-service-client';
import { ConnectError } from '@connectrpc/connect';
import { create, toJsonString } from '@bufbuild/protobuf';
import { toast } from 'svelte-sonner';

// The action to send to a service: write attribute values or invoke a command.
export type ServiceActionBody =
	| { case: 'write'; values: FieldValue[] }
	| { case: 'command'; id: string; params: FieldValue[] };

/**
 * Sends an action to one service and passes every response (including
 * intermediate QUEUED/SENT ones) to responseHandler. A transport error is
 * reported to the handler as a final ERROR response.
 */
export const SendActionRequest = async (
	deviceID: string,
	endpointID: string,
	serviceID: string,
	body: ServiceActionBody,
	responseHandler: (response: ActionResponse) => void
) => {
	const request = create(ActionRequestSchema, {
		deviceId: deviceID,
		endpointId: endpointID,
		serviceId: serviceID,
		action:
			body.case === 'write'
				? { case: 'write', value: create(WriteActionSchema, { values: body.values }) }
				: { case: 'command', value: create(CommandActionSchema, { id: body.id, params: body.params }) }
	});
	console.log('sending action: ' + toJsonString(ActionRequestSchema, request));
	try {
		for await (const response of UserServiceClient.sendAction(request)) {
			console.log('received action: ' + toJsonString(ActionResponseSchema, response));
			responseHandler(response);
		}
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error action: ' + err.message);
			const response = create(ActionResponseSchema, {});
			response.status = ActionResponse_ActionStatus.ERROR;
			response.details = err.message;
			responseHandler(response);
		}
	}
};

/** True for a status that ends the action in failure. */
export const isActionFailure = (status: ActionResponse_ActionStatus): boolean =>
	status >= ActionResponse_ActionStatus.TIMEOUT;

const failureTitle = (status: ActionResponse_ActionStatus): string => {
	switch (status) {
		case ActionResponse_ActionStatus.TIMEOUT:
			return 'Action Timed Out';
		case ActionResponse_ActionStatus.CANCELED:
			return 'Action Canceled';
		case ActionResponse_ActionStatus.ERR_READONLY:
			return 'Read-Only Attribute';
		case ActionResponse_ActionStatus.ERR_INVALID:
			return 'Invalid Action';
	}
	return 'Action Failed';
};

const failureDetails = (response: ActionResponse): string => {
	if (response.details !== '') return response.details;
	switch (response.status) {
		case ActionResponse_ActionStatus.TIMEOUT:
			return 'The device did not respond in time.';
		case ActionResponse_ActionStatus.ERR_READONLY:
			return 'The action wrote to a read-only attribute.';
		case ActionResponse_ActionStatus.ERR_INVALID:
			return 'The action does not match what the service accepts.';
	}
	return '';
};

// Sends an action and waits for its final response. Resolves with the result
// fields on COMPLETE (an empty list when the command returns nothing), or
// undefined when the action failed; the failure has then already been shown
// as a toast.
const runAction = async (
	deviceID: string,
	endpointID: string,
	serviceID: string,
	body: ServiceActionBody
): Promise<FieldValue[] | undefined> => {
	const outcome: { final?: ActionResponse } = {};
	await SendActionRequest(deviceID, endpointID, serviceID, body, (response) => {
		if (response.status >= ActionResponse_ActionStatus.COMPLETE) {
			outcome.final = response;
		}
	});
	const final = outcome.final;
	if (final === undefined) {
		toast.error('Action Failed', { description: 'No response from the server.' });
		return undefined;
	}
	if (isActionFailure(final.status)) {
		console.error('action failed:', final);
		toast.error(failureTitle(final.status), { description: failureDetails(final) });
		return undefined;
	}
	return final.result;
};

/**
 * Writes attribute values of a service. Resolves with the result fields, or
 * undefined if the write failed (ERR_READONLY, ERR_INVALID, ERROR, TIMEOUT,
 * CANCELED), after showing the failure as a toast.
 */
export const sendWrite = (
	deviceID: string,
	endpointID: string,
	serviceID: string,
	fields: FieldValue[]
): Promise<FieldValue[] | undefined> =>
	runAction(deviceID, endpointID, serviceID, { case: 'write', values: fields });

/**
 * Invokes a command of a service. Resolves with the command's result fields,
 * or undefined if it failed, after showing the failure as a toast.
 */
export const sendCommand = (
	deviceID: string,
	endpointID: string,
	serviceID: string,
	commandID: string,
	params: FieldValue[] = []
): Promise<FieldValue[] | undefined> =>
	runAction(deviceID, endpointID, serviceID, { case: 'command', id: commandID, params });

// Sets the user's label for an endpoint. An empty label clears it, so the
// bridge's label shows again.
export const SetEndpointLabel = async (
	deviceID: string,
	endpointID: string,
	label: string
): Promise<null | ConnectError> => {
	const request = create(SetEndpointLabelRequestSchema, { deviceId: deviceID, endpointId: endpointID, label });
	console.log('sending set endpoint label: ' + toJsonString(SetEndpointLabelRequestSchema, request));
	try {
		await UserServiceClient.setEndpointLabel(request);
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error set endpoint label: ' + err.message);
			return err;
		}
		throw err;
	}
	return null;
};

export const SendRemoveDeviceRequest = async (deviceID: string): Promise<null | ConnectError> => {
	const request = create(RemoveDeviceRequestSchema, {
		deviceId: deviceID
	});
	console.log('sending remove device request: ' + toJsonString(RemoveDeviceRequestSchema, request));
	try {
		await UserServiceClient.removeDevice(request);
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error sending remove device request: ' + err.message);
			return err;
		}
	}
	return null;
};

// A favorite is identified by device, endpoint and service.
export const SendFavoriteRequest = async (deviceID: string, endpointID: string, serviceID: string, fave: boolean) => {
	if (fave) {
		const request = create(AddFavoriteRequestSchema, {
			deviceId: deviceID,
			endpointId: endpointID,
			serviceId: serviceID
		});
		console.log('sending add favorite request: ' + toJsonString(AddFavoriteRequestSchema, request));
		try {
			await UserServiceClient.addFavorite(request);
		} catch (err) {
			if (err instanceof ConnectError) {
				console.error('error sending add favorite request: ' + err.message);
			}
		}
	} else {
		const request = create(RemoveFavoriteRequestSchema, {
			deviceId: deviceID,
			endpointId: endpointID,
			serviceId: serviceID
		});
		console.log('sending remove favorite request: ' + toJsonString(RemoveFavoriteRequestSchema, request));
		try {
			await UserServiceClient.removeFavorite(request);
		} catch (err) {
			if (err instanceof ConnectError) {
				console.error('error sending remove favorite request: ' + err.message);
			}
		}
	}
};

export const AddUser = async (
	username: string,
	fullname: string,
	role: UserRole,
	initialPassword: string
): Promise<null | ConnectError> => {
	const request = create(AddUserRequestSchema, {
		username: username,
		fullname: fullname,
		role: role,
		initialPassword: initialPassword
	});
	const redacted = create(AddUserRequestSchema, {
		username: username,
		fullname: fullname,
		role: role
	});
	console.log('sending add user: ' + toJsonString(AddUserRequestSchema, redacted));
	try {
		const response = await UserServiceClient.addUser(request);
		console.log('received add user: ' + toJsonString(AddUserResponseSchema, response));
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error add user: ' + err.message);
			return err;
		}
	}
	return null;
};

export const UpdateUserFullname = async (username: string, fullname: string): Promise<null | ConnectError> => {
	return UpdateUser(
		create(UpdateUserRequestSchema, {
			username: username,
			fullname: fullname
		})
	);
};

export const UpdateUserRole = async (username: string, role: UserRole): Promise<null | ConnectError> => {
	return UpdateUser(
		create(UpdateUserRequestSchema, {
			username: username,
			role: role
		})
	);
};

// Changing your own password. The current password is required by the
// server and is what stops a stolen session from locking the real owner
// out, so it is not optional here either.
export const UpdateUserPassword = async (
	username: string,
	currentPassword: string,
	newPassword: string
): Promise<null | ConnectError> => {
	return UpdateUser(
		create(UpdateUserRequestSchema, {
			username: username,
			currentPassword: currentPassword,
			password: newPassword
		})
	);
};

// Admin-only: set a temporary password for somebody else. The server forces
// them to choose their own on next login and drops all of their sessions.
export const ResetUserPassword = async (username: string, newPassword: string): Promise<null | ConnectError> => {
	return UpdateUser(
		create(UpdateUserRequestSchema, {
			username: username,
			password: newPassword
		})
	);
};

export const UpdateUser = async (request: UpdateUserRequest): Promise<null | ConnectError> => {
	// Log a copy with the password fields dropped - this request carries
	// cleartext passwords on the change-password paths, and the console is
	// not the place for them. Same treatment AddUser gives initialPassword.
	const redacted = create(UpdateUserRequestSchema, {
		...request,
		password: request.password === undefined ? undefined : '<redacted>',
		currentPassword: request.currentPassword === undefined ? undefined : '<redacted>'
	});
	console.log('sending update user: ' + toJsonString(UpdateUserRequestSchema, redacted));
	try {
		const response = await UserServiceClient.updateUser(request);
		console.log('received update user: ' + toJsonString(UpdateUserResponseSchema, response));
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error update user: ' + err.message);
			return err;
		}
	}
	return null;
};

export const ApprovePairing = async (clientID: string, requestID: string): Promise<null | ConnectError> => {
	const request = create(ApprovePairingRequestSchema, {
		clientId: clientID,
		requestId: requestID
	});
	console.log('sending approve pairing: ' + toJsonString(ApprovePairingRequestSchema, request));
	try {
		const response = await UserServiceClient.approvePairing(request);
		console.log('received approve pairing: ' + toJsonString(ApprovePairingResponseSchema, response));
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error approve pairing: ' + err.message);
			return err;
		}
	}
	return null;
};

export const DenyPairing = async (clientID: string, requestID: string): Promise<null | ConnectError> => {
	const request = create(DenyPairingRequestSchema, {
		clientId: clientID,
		requestId: requestID
	});
	console.log('sending deny pairing: ' + toJsonString(DenyPairingRequestSchema, request));
	try {
		const response = await UserServiceClient.denyPairing(request);
		console.log('received deny pairing: ' + toJsonString(DenyPairingResponseSchema, response));
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error deny pairing: ' + err.message);
			return err;
		}
	}
	return null;
};

export const UnpairClient = async (clientID: string): Promise<null | ConnectError> => {
	const request = create(UnpairClientRequestSchema, {
		clientId: clientID
	});
	console.log('sending unpair client: ' + toJsonString(UnpairClientRequestSchema, request));
	try {
		const response = await UserServiceClient.unpairClient(request);
		console.log('received unpair client: ' + toJsonString(UnpairClientResponseSchema, response));
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error unpair client: ' + err.message);
			return err;
		}
	}
	return null;
};

export const AddGroup = async (
	name: string,
	type: Service_ServiceType,
	members: GroupMember[]
): Promise<null | ConnectError> => {
	const request = create(AddGroupRequestSchema, { name, type, members });
	console.log('sending add group: ' + toJsonString(AddGroupRequestSchema, request));
	try {
		const response = await UserServiceClient.addGroup(request);
		console.log('received add group: ' + toJsonString(AddGroupResponseSchema, response));
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error add group: ' + err.message);
			return err;
		}
	}
	return null;
};

export const UpdateGroup = async (id: string, name?: string, members?: GroupMember[]): Promise<null | ConnectError> => {
	const request = create(UpdateGroupRequestSchema, { id, name, members: members ?? [] });
	console.log('sending update group: ' + toJsonString(UpdateGroupRequestSchema, request));
	try {
		const response = await UserServiceClient.updateGroup(request);
		console.log('received update group: ' + toJsonString(UpdateGroupResponseSchema, response));
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error update group: ' + err.message);
			return err;
		}
	}
	return null;
};

export const RemoveGroup = async (id: string): Promise<null | ConnectError> => {
	const request = create(RemoveGroupRequestSchema, { id });
	console.log('sending remove group: ' + toJsonString(RemoveGroupRequestSchema, request));
	try {
		const response = await UserServiceClient.removeGroup(request);
		console.log('received remove group: ' + toJsonString(RemoveGroupResponseSchema, response));
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error remove group: ' + err.message);
			return err;
		}
	}
	return null;
};

// `endpoints` are endpoint overrides: endpoints that belong to this zone
// although their device is in another zone (or none).
export const AddZone = async (
	name: string,
	icon: string,
	deviceIds: string[],
	endpoints: EndpointRef[] = []
): Promise<null | ConnectError> => {
	const request = create(AddZoneRequestSchema, { name, icon, deviceIds, endpoints });
	console.log('sending add zone: ' + toJsonString(AddZoneRequestSchema, request));
	try {
		const response = await UserServiceClient.addZone(request);
		console.log('received add zone: ' + toJsonString(AddZoneResponseSchema, response));
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error add zone: ' + err.message);
			return err;
		}
	}
	return null;
};

// Membership is only sent when `deviceIds` is passed, and then it replaces the
// zone's devices in full. Passing an empty array empties the zone - that is why
// the request carries an explicit setDevices flag rather than treating an empty
// list as "leave it alone", the way UpdateGroup does. Endpoint overrides work
// the same way through `endpoints` and setEndpoints.
export const UpdateZone = async (
	id: string,
	changes: { name?: string; icon?: string; deviceIds?: string[]; endpoints?: EndpointRef[] }
): Promise<null | ConnectError> => {
	const request = create(UpdateZoneRequestSchema, {
		id,
		name: changes.name,
		icon: changes.icon,
		setDevices: changes.deviceIds !== undefined,
		deviceIds: changes.deviceIds ?? [],
		setEndpoints: changes.endpoints !== undefined,
		endpoints: changes.endpoints ?? []
	});
	console.log('sending update zone: ' + toJsonString(UpdateZoneRequestSchema, request));
	try {
		const response = await UserServiceClient.updateZone(request);
		console.log('received update zone: ' + toJsonString(UpdateZoneResponseSchema, response));
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error update zone: ' + err.message);
			return err;
		}
	}
	return null;
};

export const RemoveZone = async (id: string): Promise<null | ConnectError> => {
	const request = create(RemoveZoneRequestSchema, { id });
	console.log('sending remove zone: ' + toJsonString(RemoveZoneRequestSchema, request));
	try {
		const response = await UserServiceClient.removeZone(request);
		console.log('received remove zone: ' + toJsonString(RemoveZoneResponseSchema, response));
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error remove zone: ' + err.message);
			return err;
		}
	}
	return null;
};

export const SendImageRequest = async (
	deviceID: string,
	endpointID: string,
	serviceID: string,
	attributeID: string,
	responseHandler: (response: ImageResponse) => void,
	width = 0,
	height = 0
) => {
	const request = create(UserImageRequestSchema, {
		deviceId: deviceID,
		endpointId: endpointID,
		serviceId: serviceID,
		attributeId: attributeID,
		width,
		height
	});
	console.log(
		'sending image request: deviceId=' + deviceID + ' endpointId=' + endpointID + ' serviceId=' + serviceID
	);
	try {
		for await (const response of UserServiceClient.sendImageRequest(request)) {
			console.log('received image response: status=' + response.status + ' dataLen=' + response.data.length);
			responseHandler(response);
			if (response.status >= ImageResponse_ImageStatus.COMPLETE) {
				break;
			}
		}
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error image request: ' + err.message);
			const response = create(ImageResponseSchema, {
				status: ImageResponse_ImageStatus.ERROR,
				details: err.message
			});
			responseHandler(response);
		}
	}
};

export const ForgetClient = async (clientID: string): Promise<null | ConnectError> => {
	const request = create(ForgetClientRequestSchema, {
		clientId: clientID
	});
	console.log('sending forget client: ' + toJsonString(ForgetClientRequestSchema, request));
	try {
		const response = await UserServiceClient.forgetClient(request);
		console.log('received forget client: ' + toJsonString(ForgetClientResponseSchema, response));
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error forget client: ' + err.message);
			return err;
		}
	}
	return null;
};

export type ServerSettings = {
	instanceName: string;
	showInstanceName: boolean;
};

const settingsFrom = (settings: { instanceName?: string; showInstanceName?: boolean } | undefined): ServerSettings => ({
	instanceName: settings?.instanceName ?? '',
	showInstanceName: settings?.showInstanceName ?? false
});

export const GetSettings = async (): Promise<ServerSettings | ConnectError> => {
	const request = create(GetSettingsRequestSchema, {});
	console.log('sending get settings: ' + toJsonString(GetSettingsRequestSchema, request));
	try {
		const response = await UserServiceClient.getSettings(request);
		return settingsFrom(response.settings);
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error get settings: ' + err.message);
			return err;
		}
		throw err;
	}
};

// Only the fields present in update are changed. Returns the settings as
// stored, which may differ from what was asked for because the server trims
// the instance name.
export const UpdateSettings = async (update: {
	instanceName?: string;
	showInstanceName?: boolean;
}): Promise<ServerSettings | ConnectError> => {
	const request = create(UpdateSettingsRequestSchema, update);
	console.log('sending update settings: ' + toJsonString(UpdateSettingsRequestSchema, request));
	try {
		const response = await UserServiceClient.updateSettings(request);
		return settingsFrom(response.settings);
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error update settings: ' + err.message);
			return err;
		}
		throw err;
	}
};

export const MarkNotificationRead = async (id: string): Promise<null | ConnectError> => {
	const request = create(MarkNotificationReadRequestSchema, { id });
	console.log('sending mark notification read: ' + toJsonString(MarkNotificationReadRequestSchema, request));
	try {
		await UserServiceClient.markNotificationRead(request);
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error mark notification read: ' + err.message);
			return err;
		}
		throw err;
	}
	return null;
};

export const MarkAllNotificationsRead = async (): Promise<null | ConnectError> => {
	const request = create(MarkAllNotificationsReadRequestSchema, {});
	console.log('sending mark all notifications read');
	try {
		await UserServiceClient.markAllNotificationsRead(request);
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error mark all notifications read: ' + err.message);
			return err;
		}
		throw err;
	}
	return null;
};

export const DismissNotification = async (id: string): Promise<null | ConnectError> => {
	const request = create(DismissNotificationRequestSchema, { id });
	console.log('sending dismiss notification: ' + toJsonString(DismissNotificationRequestSchema, request));
	try {
		await UserServiceClient.dismissNotification(request);
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error dismiss notification: ' + err.message);
			return err;
		}
		throw err;
	}
	return null;
};

// Raises a notification for the given audience. The only notification trigger
// there is for now - everything else that will eventually raise one (devices,
// bridges, reactors, the core itself) goes through the same manager.
export const SendTestNotification = async (test: {
	title?: string;
	body?: string;
	level?: NotificationLevel;
	audience?: { kind: NotificationAudience_Kind; role?: UserRole; username?: string };
}): Promise<Notification | ConnectError> => {
	const request = create(SendTestNotificationRequestSchema, test);
	console.log('sending test notification: ' + toJsonString(SendTestNotificationRequestSchema, request));
	try {
		const response = await UserServiceClient.sendTestNotification(request);
		if (response.notification === undefined) {
			return new ConnectError('server returned no notification');
		}
		return response.notification;
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error send test notification: ' + err.message);
			return err;
		}
		throw err;
	}
};
