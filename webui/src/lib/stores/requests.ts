import { ActionRequestSchema, ActionResponseSchema, ActionResponse_ActionStatus, type Value, type ActionResponse } from '$lib/api/v1/clients/client_service_pb';
import { AddFavoriteRequestSchema, AddFavoriteResponseSchema, RemoveFavoriteRequestSchema } from '$lib/api/v1/clients/user_service_pb';
import { UserServiceClient } from './user-service-client';
import { ConnectError } from '@connectrpc/connect';
import { create, toJsonString } from "@bufbuild/protobuf";

export const SendActionRequest = async (deviceID: string, serviceID: string, vals: Value[], responseHandler: (response: ActionResponse) => void) => {
	const request = create(ActionRequestSchema, {
		deviceId: deviceID,
		serviceId: serviceID,
		values: vals
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

export const SendFavoriteRequest = async (deviceID: string, serviceID: string, fave: boolean) => {
	if (fave) {
		const request = create(AddFavoriteRequestSchema, {
			deviceId: deviceID,
			serviceId: serviceID,
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
			serviceId: serviceID,
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
