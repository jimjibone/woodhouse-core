import { writable } from 'svelte/store';
import type * as grpcWeb from 'grpc-web';
import { DeviceState, type DeviceExtendedInfo, type DeviceResponse } from '../api/device_pb';
import type { DeviceRequest } from '../api/device_pb';
import { ReactorServiceClient } from '../api/Reactor_serviceServiceClientPb';
import { GetDeviceInfosRequest, GetDeviceStatesRequest, SetDeviceFavouriteRequest, SetDeviceFavouriteResponse, SetDeviceHiddenRequest, SetDeviceHiddenResponse } from '../api/reactor_service_pb';
import { defaultMinBackoffMs, defaultMaxBackoffMs, createBackoffWithHeartbeat } from './utils';

const reactorClient = new ReactorServiceClient('/api');
const debug = false;

export const deviceInfosStream = createDeviceInfosStream("getDeviceInfos", debug);
function createDeviceInfosStream(name: string, debug: boolean) {
	let data: DeviceExtendedInfo[] = [];
	let connected: boolean = false;
	let stream: grpcWeb.ClientReadableStream<DeviceExtendedInfo>|null = null;
	const dataWriter = writable(data, start);
	const connectedWriter = writable(connected);

	const backoff = createBackoffWithHeartbeat(name, defaultMinBackoffMs, defaultMaxBackoffMs, 60000, run, stop);

	function start() : VoidFunction {
		if (debug) console.log(`${name}: starting...`);
		backoff.start();
		return backoff.stop;
	}

	function stop() {
		if (debug) console.log(`${name}: stopping`);
		if (stream != null) {
			stream.cancel();
			stream = null;
		}
	}

	function run(resetHeartbeat: VoidFunction, restartConnection: VoidFunction) {
		if (debug) console.log(`${name}: started`);
		const request = new GetDeviceInfosRequest();
		stream = reactorClient.getDeviceInfos(request);
		stream.on("error", (err: grpcWeb.RpcError) => {
			console.error(`${name}: unexpected stream error: code = ${err.code}` + `, message = "${err.message}"`);
			connectedWriter.set(false);
			restartConnection();
		});
		stream.on("data", (response: DeviceExtendedInfo) => {
			if (debug) console.log(`${name}: data:`, response.toObject());
			connectedWriter.set(true);
			resetHeartbeat();
			if (response.getBridgeId() !== "") {
				dataWriter.update(u => {
					const response_id = response.getBridgeId() + "." + response.getDeviceId();
					let updated = false;
					for (let i = 0; i < u.length; i++) {
						const u_id = u[i].getBridgeId() + "." + u[i].getDeviceId();
						if (u_id === response_id) {
							updated = true;
							u[i] = response;
							break;
						}
					}
					if (!updated) u = [...u, response];
					return u
				});
			}
		});
		stream.on("end", () => {
			if (debug) console.log(`${name}: done`);
			connectedWriter.set(false);
			restartConnection();
		});
	}

	return {
		subscribeData: dataWriter.subscribe,
		subscribeConnected: connectedWriter.subscribe,
	};
}

export const deviceStatesStream = createDeviceStatesStream("getDeviceStates", debug);
function createDeviceStatesStream(name: string, debug: boolean) {
	let data: DeviceState[] = [];
	let connected: boolean = false;
	let stream: grpcWeb.ClientReadableStream<DeviceState>|null = null;
	const dataWriter = writable(data, start);
	const connectedWriter = writable(connected);

	const backoff = createBackoffWithHeartbeat(name, defaultMinBackoffMs, defaultMaxBackoffMs, 60000, run, stop);

	function start() : VoidFunction {
		if (debug) console.log(`${name}: starting...`);
		backoff.start();
		return backoff.stop;
	}

	function stop() {
		if (debug) console.log(`${name}: stopping`);
		if (stream != null) {
			stream.cancel();
			stream = null;
		}
	}

	function run(resetHeartbeat: VoidFunction, restartConnection: VoidFunction) {
		if (debug) console.log(`${name}: started`);
		const request = new GetDeviceStatesRequest();
		stream = reactorClient.getDeviceStates(request);
		stream.on("error", (err: grpcWeb.RpcError) => {
			console.error(`${name}: unexpected stream error: code = ${err.code}` + `, message = "${err.message}"`);
			connectedWriter.set(false);
			restartConnection();
		});
		stream.on("data", (response: DeviceState) => {
			if (debug) console.log(`${name}: data:`, response.toObject());
			connectedWriter.set(true);
			resetHeartbeat();
			if (response.getBridgeId() !== "") {
				dataWriter.update(u => {
					const response_id = response.getBridgeId() + "." + response.getDeviceId();
					let updated = false;
					for (let i = 0; i < u.length; i++) {
						const u_id = u[i].getBridgeId() + "." + u[i].getDeviceId();
						if (u_id === response_id) {
							updated = true;
							u[i] = response;
							break;
						}
					}
					if (!updated) u = [...u, response];
					return u
				});
			}
		});
		stream.on("end", () => {
			if (debug) console.log(`${name}: done`);
			connectedWriter.set(false);
			restartConnection();
		});
	}

	return {
		subscribeData: dataWriter.subscribe,
		subscribeConnected: connectedWriter.subscribe,
	};
}

export type DeviceInfoState = {
	fullId: string,
	info: DeviceExtendedInfo|null,
	state: DeviceState|null,
}

export const devicesStream = createDevicesStream("deviceStream", debug);
function createDevicesStream(name: string, debug: boolean) {
	let data: DeviceInfoState[] = [];
	let connected: boolean = false;
	const dataWriter = writable(data, start);
	const connectedWriter = writable(connected);

	let infoStream: grpcWeb.ClientReadableStream<DeviceExtendedInfo>|null = null;
	let stateStream: grpcWeb.ClientReadableStream<DeviceState>|null = null;

	const backoff = createBackoffWithHeartbeat(name, defaultMinBackoffMs, defaultMaxBackoffMs, 60000, run, stop);

	function start() : VoidFunction {
		if (debug) console.log(`${name}: starting...`);
		backoff.start();
		return backoff.stop;
	}

	function stop() {
		if (debug) console.log(`${name}: stopping`);
		if (infoStream != null) {
			infoStream.cancel();
			infoStream = null;
		}
		if (stateStream != null) {
			stateStream.cancel();
			stateStream = null;
		}
	}

	function run(resetHeartbeat: VoidFunction, restartConnection: VoidFunction) {
		if (debug) console.log(`${name}: started`);
		// DeviceInfo
		infoStream = reactorClient.getDeviceInfos(new GetDeviceInfosRequest());
		infoStream.on("error", (err: grpcWeb.RpcError) => {
			console.error(`${name}: unexpected info stream error: code = ${err.code}` + `, message = "${err.message}"`);
			connectedWriter.set(false);
			if (infoStream) infoStream.cancel();
			if (stateStream) stateStream.cancel();
			restartConnection();
		});
		infoStream.on("data", (response: DeviceExtendedInfo) => {
			if (debug) console.log(`${name}: info data:`, response.toObject());
			connectedWriter.set(true);
			resetHeartbeat();
			if (response.getBridgeId() !== "") {
				dataWriter.update(devices => {
					const response_id = response.getBridgeId() + "." + response.getDeviceId();
					let updated = false;
					for (let i = 0; i < devices.length; i++) {
						if (devices[i].fullId === response_id) {
							updated = true;
							devices[i].info = response;
							break;
						}
					}
					if (!updated) devices = [...devices, {
						fullId: response_id,
						info: response,
						state: null,
					}];
					return devices;
				});
			}
		});
		infoStream.on("end", () => {
			if (debug) console.log(`${name}: info done`);
			connectedWriter.set(false);
			if (infoStream) infoStream.cancel();
			if (stateStream) stateStream.cancel();
			restartConnection();
		});
		// DeviceState
		stateStream = reactorClient.getDeviceStates(new GetDeviceStatesRequest());
		stateStream.on("error", (err: grpcWeb.RpcError) => {
			console.error(`${name}: unexpected state stream error: code = ${err.code}` + `, message = "${err.message}"`);
			connectedWriter.set(false);
			if (infoStream) infoStream.cancel();
			if (stateStream) stateStream.cancel();
			restartConnection();
		});
		stateStream.on("data", (response: DeviceState) => {
			if (debug) console.log(`${name}: state data:`, response.toObject());
			connectedWriter.set(true);
			resetHeartbeat();
			if (response.getBridgeId() !== "") {
				dataWriter.update(devices => {
					const response_id = response.getBridgeId() + "." + response.getDeviceId();
					let updated = false;
					for (let i = 0; i < devices.length; i++) {
						if (devices[i].fullId === response_id) {
							updated = true;
							devices[i].state = response;
							break;
						}
					}
					if (!updated) devices = [...devices, {
						fullId: response_id,
						info: null,
						state: response,
					}];
					return devices;
				});
			}
		});
		stateStream.on("end", () => {
			if (debug) console.log(`${name}: state done`);
			connectedWriter.set(false);
			if (infoStream) infoStream.cancel();
			if (stateStream) stateStream.cancel();
			restartConnection();
		});
	}

	return {
		subscribe: dataWriter.subscribe,
		subscribeConnected: connectedWriter.subscribe,
	};
}

export function setDeviceHidden(req: SetDeviceHiddenRequest) : Promise<SetDeviceHiddenResponse> {
	return reactorClient.setDeviceHidden(req, null); //, {"authorization": getAccessToken()});
}

export function setDeviceFavourite(req: SetDeviceFavouriteRequest) : Promise<SetDeviceFavouriteResponse> {
	return reactorClient.setDeviceFavourite(req, null); //, {"authorization": getAccessToken()});
}

export function sendDeviceRequest(req: DeviceRequest) : Promise<DeviceResponse> {
	return reactorClient.sendDeviceRequest(req, null); //, {"authorization": getAccessToken()});
}
