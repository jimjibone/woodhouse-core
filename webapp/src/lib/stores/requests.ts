import { ActionRequest, ActionResponse, ActionResponse_ActionStatus, ImageRequest, ImageResponse_ImageStatus, Value } from '$lib/api/v1/clients/client_service_pb';
import { AddFavoriteRequest, RemoveFavoriteRequest } from '@/api/v1/clients/user_service_pb';
import { UserServiceClient } from './user-service-client';
import { ConnectError } from '@connectrpc/connect';

export const SendActionRequest = async (deviceID: string, serviceID: string, vals: Value[], responseHandler: (response: ActionResponse) => void) => {
	const request = new ActionRequest({
		deviceId: deviceID,
		serviceId: serviceID,
		values: vals
	});
	console.log('sending action: ' + request.toJsonString());
	try {
		for await (const response of UserServiceClient.sendAction(request)) {
			console.log('received action: ' + response.toJsonString());
			responseHandler(response);
		}
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error action: ' + err.message);
			const response = new ActionResponse();
			response.status = ActionResponse_ActionStatus.ERROR;
			response.details = err.message;
			responseHandler(response);
		}
	}
};

export const SendImageRequest = async (deviceID: string, serviceID: string, attributeID: string, dataHandler: (data: Uint8Array) => void, errHandler: (status: ImageResponse_ImageStatus, details: string) => void) => {
	const request = new ImageRequest({
		deviceId: deviceID,
		serviceId: serviceID,
		attributeId: attributeID
	});
	console.log('sending image request: ' + request.toJsonString());
	try {
		for await (const response of UserServiceClient.sendImageRequest(request)) {
			console.log('received image response: request-id:' + response.requestId + ', status:' + ImageResponse_ImageStatus[response.status] + ', details:' + response.details + ', data:' + response.data.length + ' bytes');
			if (response.status >= ImageResponse_ImageStatus.TIMEOUT) {
				errHandler(response.status, response.details);
			} else if (response.data.length > 0) {
				dataHandler(response.data);
			}
		}
	} catch (err) {
		if (err instanceof ConnectError) {
			console.error('error image request: ' + err.message);
		}
	}
};

export const SendFavoriteRequest = async (deviceID: string, serviceID: string, fave: boolean) => {
	if (fave) {
		const request = new AddFavoriteRequest({
			deviceId: deviceID,
			serviceId: serviceID,
		});
		console.log('sending add favorite request: ' + request.toJsonString());
		try {
			await UserServiceClient.addFavorite(request);
		} catch (err) {
			if (err instanceof ConnectError) {
				console.error('error sending add favorite request: ' + err.message);
			}
		}
	} else {
		const request = new RemoveFavoriteRequest({
			deviceId: deviceID,
			serviceId: serviceID,
		});
		console.log('sending remove favorite request: ' + request.toJsonString());
		try {
			await UserServiceClient.removeFavorite(request);
		} catch (err) {
			if (err instanceof ConnectError) {
				console.error('error sending remove favorite request: ' + err.message);
			}
		}
	}
};
