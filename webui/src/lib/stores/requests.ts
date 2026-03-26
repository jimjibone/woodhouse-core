import {
	ActionRequestSchema,
	ActionResponseSchema,
	ActionResponse_ActionStatus,
	type Value,
	type ActionResponse,
	Service_ServiceType
} from '$lib/api/v1/clients/client_service_pb';
import {
	AddFavoriteRequestSchema,
	AddGroupRequestSchema,
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
	UpdateUserRequestSchema,
	UpdateUserResponseSchema,
	UserRole,
	type UpdateUserRequest
} from '$lib/api/v1/clients/user_service_pb';
import { type GroupMember } from '$lib/api/v1/clients/group_pb';
import { UserServiceClient } from './user-service-client';
import { ConnectError, type CallOptions } from '@connectrpc/connect';
import { create, toJsonString } from '@bufbuild/protobuf';
import { getAccessToken } from '$lib/stores/auth-store';

export const SendActionRequest = async (
	deviceID: string,
	serviceID: string,
	vals: Value[],
	responseHandler: (response: ActionResponse) => void
) => {
	const request = create(ActionRequestSchema, {
		deviceId: deviceID,
		serviceId: serviceID,
		values: vals
	});
	const options: CallOptions = {
		headers: { authorization: getAccessToken() }
	};
	console.log('sending action: ' + toJsonString(ActionRequestSchema, request));
	try {
		for await (const response of UserServiceClient.sendAction(request, options)) {
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

export const SendRemoveDeviceRequest = async (deviceID: string): Promise<null | ConnectError> => {
	const request = create(RemoveDeviceRequestSchema, {
		deviceId: deviceID
	});
	const options: CallOptions = {
		headers: { authorization: getAccessToken() }
	};
	console.log('sending remove device request: ' + toJsonString(RemoveDeviceRequestSchema, request));
	try {
		await UserServiceClient.removeDevice(request, options);
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error sending remove device request: ' + err.message);
			return err;
		}
	}
	return null;
};

export const SendFavoriteRequest = async (deviceID: string, serviceID: string, fave: boolean) => {
	if (fave) {
		const request = create(AddFavoriteRequestSchema, {
			deviceId: deviceID,
			serviceId: serviceID
		});
		const options: CallOptions = {
			headers: { authorization: getAccessToken() }
		};
		console.log('sending add favorite request: ' + toJsonString(AddFavoriteRequestSchema, request));
		try {
			await UserServiceClient.addFavorite(request, options);
		} catch (err) {
			if (err instanceof ConnectError) {
				console.error('error sending add favorite request: ' + err.message);
			}
		}
	} else {
		const request = create(RemoveFavoriteRequestSchema, {
			deviceId: deviceID,
			serviceId: serviceID
		});
		const options: CallOptions = {
			headers: { authorization: getAccessToken() }
		};
		console.log('sending remove favorite request: ' + toJsonString(RemoveFavoriteRequestSchema, request));
		try {
			await UserServiceClient.removeFavorite(request, options);
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
	const options: CallOptions = {
		headers: { authorization: getAccessToken() }
	};
	console.log('sending add user: ' + toJsonString(AddUserRequestSchema, redacted));
	try {
		const response = await UserServiceClient.addUser(request, options);
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

export const UpdateUser = async (request: UpdateUserRequest): Promise<null | ConnectError> => {
	const options: CallOptions = {
		headers: { authorization: getAccessToken() }
	};
	console.log('sending update user: ' + toJsonString(UpdateUserRequestSchema, request));
	try {
		const response = await UserServiceClient.updateUser(request, options);
		console.log('received update user: ' + toJsonString(UpdateUserResponseSchema, response));
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error update user: ' + err.message);
			return err;
		}
	}
	return null;
};

export const ApprovePairing = async (clientID: string, pairingCode: string): Promise<null | ConnectError> => {
	const request = create(ApprovePairingRequestSchema, {
		clientId: clientID,
		pairingCode: pairingCode.trim()
	});
	const options: CallOptions = {
		headers: { authorization: getAccessToken() }
	};
	console.log('sending approve pairing: ' + toJsonString(ApprovePairingRequestSchema, request));
	try {
		const response = await UserServiceClient.approvePairing(request, options);
		console.log('received approve pairing: ' + toJsonString(ApprovePairingResponseSchema, response));
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error approve pairing: ' + err.message);
			return err;
		}
	}
	return null;
};

export const DenyPairing = async (clientID: string): Promise<null | ConnectError> => {
	const request = create(DenyPairingRequestSchema, {
		clientId: clientID
	});
	const options: CallOptions = {
		headers: { authorization: getAccessToken() }
	};
	console.log('sending deny pairing: ' + toJsonString(DenyPairingRequestSchema, request));
	try {
		const response = await UserServiceClient.denyPairing(request, options);
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
	const options: CallOptions = {
		headers: { authorization: getAccessToken() }
	};
	console.log('sending unpair client: ' + toJsonString(UnpairClientRequestSchema, request));
	try {
		const response = await UserServiceClient.unpairClient(request, options);
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
	const options: CallOptions = {
		headers: { authorization: getAccessToken() }
	};
	console.log('sending add group: ' + toJsonString(AddGroupRequestSchema, request));
	try {
		const response = await UserServiceClient.addGroup(request, options);
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
	const options: CallOptions = {
		headers: { authorization: getAccessToken() }
	};
	console.log('sending update group: ' + toJsonString(UpdateGroupRequestSchema, request));
	try {
		const response = await UserServiceClient.updateGroup(request, options);
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
	const options: CallOptions = {
		headers: { authorization: getAccessToken() }
	};
	console.log('sending remove group: ' + toJsonString(RemoveGroupRequestSchema, request));
	try {
		const response = await UserServiceClient.removeGroup(request, options);
		console.log('received remove group: ' + toJsonString(RemoveGroupResponseSchema, response));
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error remove group: ' + err.message);
			return err;
		}
	}
	return null;
};

export const ForgetClient = async (clientID: string): Promise<null | ConnectError> => {
	const request = create(ForgetClientRequestSchema, {
		clientId: clientID
	});
	const options: CallOptions = {
		headers: { authorization: getAccessToken() }
	};
	console.log('sending forget client: ' + toJsonString(ForgetClientRequestSchema, request));
	try {
		const response = await UserServiceClient.forgetClient(request, options);
		console.log('received block client: ' + toJsonString(ForgetClientResponseSchema, response));
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error unblock client: ' + err.message);
			return err;
		}
	}
	return null;
};
