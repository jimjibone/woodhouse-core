import { ActionRequestSchema, ActionResponseSchema, ActionResponse_ActionStatus, type Value, type ActionResponse } from '$lib/api/v1/clients/client_service_pb';
import { AddFavoriteRequestSchema, RemoveFavoriteRequestSchema } from '$lib/api/v1/clients/user_service_pb';
import { UserServiceClient } from './user-service-client';
import { ConnectError, type CallOptions } from '@connectrpc/connect';
import { create, toJsonString } from "@bufbuild/protobuf";
import { getAccessToken } from '$lib/stores/auth-store';

export const SendActionRequest = async (deviceID: string, serviceID: string, vals: Value[], responseHandler: (response: ActionResponse) => void) => {
	const request = create(ActionRequestSchema, {
		deviceId: deviceID,
		serviceId: serviceID,
		values: vals
	});
	const options: CallOptions = {
		headers: { "authorization": getAccessToken() }
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
			serviceId: serviceID,
		});
		const options: CallOptions = {
			headers: { "authorization": getAccessToken() }
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
			serviceId: serviceID,
		});
		const options: CallOptions = {
			headers: { "authorization": getAccessToken() }
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
