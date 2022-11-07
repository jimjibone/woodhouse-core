import { writable } from 'svelte/store';
import type * as grpcWeb from 'grpc-web';
import type { DeviceInfo, DeviceState } from '../api/device_pb';
import { ReactorServiceClient } from '../api/Reactor_serviceServiceClientPb';
import { GetDeviceInfosRequest, GetDeviceStatesRequest } from '../api/reactor_service_pb';
import { createBackoff, defaultMinBackoffMs, defaultMaxBackoffMs } from './utils';

const reactorClient = new ReactorServiceClient('/api');

export const deviceInfosStream = createDeviceInfosStream("getDeviceInfos", true);
function createDeviceInfosStream(name: string, debug: boolean) {
	let data: DeviceInfo[] = [];
	let connected: boolean = false;
	let stream: grpcWeb.ClientReadableStream<DeviceInfo> = null;
	const dataWriter = writable(data, start);
	const connectedWriter = writable(connected);

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

	const backoff = createBackoff(name, defaultMinBackoffMs, defaultMaxBackoffMs, run, stop);

	function run(restart: VoidFunction) {
		if (debug) console.log(`${name}: started`);
		const request = new GetDeviceInfosRequest();
		stream = reactorClient.getDeviceInfos(request);
		stream.on("error", (err: grpcWeb.RpcError) => {
			console.error(`${name}: unexpected stream error: code = ${err.code}` + `, message = "${err.message}"`);
			connectedWriter.set(false);
			restart();
		});
		stream.on("data", (response: DeviceInfo) => {
			if (debug) console.log(`${name}: data:`, response.toObject());
			connectedWriter.set(true);
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
			restart();
		});
	}

	return {
		subscribeData: dataWriter.subscribe,
		subscribeConnected: connectedWriter.subscribe,
	};
}

export const deviceStatesStream = createDeviceStatesStream("getDeviceStates", true);
function createDeviceStatesStream(name: string, debug: boolean) {
	let data: DeviceState[] = [];
	let connected: boolean = false;
	let stream: grpcWeb.ClientReadableStream<DeviceState> = null;
	const dataWriter = writable(data, start);
	const connectedWriter = writable(connected);

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

	const backoff = createBackoff(name, defaultMinBackoffMs, defaultMaxBackoffMs, run, stop);

	function run(restart: VoidFunction) {
		if (debug) console.log(`${name}: started`);
		const request = new GetDeviceStatesRequest();
		stream = reactorClient.getDeviceStates(request);
		stream.on("error", (err: grpcWeb.RpcError) => {
			console.error(`${name}: unexpected stream error: code = ${err.code}` + `, message = "${err.message}"`);
			connectedWriter.set(false);
			restart();
		});
		stream.on("data", (response: DeviceState) => {
			if (debug) console.log(`${name}: data:`, response.toObject());
			connectedWriter.set(true);
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
			restart();
		});
	}

	return {
		subscribeData: dataWriter.subscribe,
		subscribeConnected: connectedWriter.subscribe,
	};
}
