import {
	ActionRequestSchema,
	ActionResponseSchema,
	ActionResponse_ActionStatus,
	type Value,
	type ActionResponse
} from '$lib/api/v1/clients/client_service_pb';
import {
	AddFavoriteRequestSchema,
	AddUserRequestSchema,
	AddUserResponseSchema,
	RemoveFavoriteRequestSchema,
	UpdateUserRequestSchema,
	UpdateUserResponseSchema,
	UserRole,
	type UpdateUserRequest
} from '$lib/api/v1/clients/user_service_pb';
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
