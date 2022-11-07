import { writable } from 'svelte/store';
import type * as grpcWeb from 'grpc-web';
import type { DeviceInfo } from './api/device_pb';
import { ReactorServiceClient } from './api/Reactor_serviceServiceClientPb';
import { GetDeviceInfosRequest } from './api/reactor_service_pb';
import { differenceInMilliseconds } from 'date-fns';

const reactorClient = new ReactorServiceClient('/api');

const defaultMinBackoffMs = 1000;
const defaultMaxBackoffMs = 30000;

// Run a function which should be restarted occasionally with some exponential
// backoff.
function runBackoff(debug: string, minBackoffMs: number, maxBackoffMs: number, run: (restart: VoidFunction) => void) {
	let backoffDurationMs = 0;
	let lastBackoffTime = new Date();

	function doRestart() {
		// Reset the backoff duration if the backoff has not been used for a
		// suitable amount of time.
		const now = new Date();
		const dt = differenceInMilliseconds(now, lastBackoffTime);
		if (dt > backoffDurationMs) {
			if (debug !== "") console.log(`${debug}: backoff reset after ${dt} ms`);
			backoffDurationMs = minBackoffMs;
		}
		lastBackoffTime = now;
		if (debug !== "") console.log(`${debug}: starting backoff for ${backoffDurationMs} ms`);
		setTimeout(() => {
			if (debug !== "") console.log(`${debug}: backoff finished`);
			backoffDurationMs = backoffDurationMs * 2
			if (backoffDurationMs > maxBackoffMs) {
				backoffDurationMs = maxBackoffMs;
			}
			run(doRestart);
		}, backoffDurationMs);
	}

	run(doRestart);
}

function createDeviceInfosStream(debug: string) {
	let data: DeviceInfo[] = [];
	let connected: boolean = false;
	const dataWriter = writable(data);
	const connectedWriter = writable(connected);

	function run(restart: VoidFunction) {
		if (debug !== "") console.log(`${debug}: starting...`);
		const request = new GetDeviceInfosRequest();
		const stream = reactorClient.getDeviceInfos(request);
		// if (debug !== "") {
		// 	stream.on("status", (status: grpcWeb.Status) => {
		// 		console.log(`${debug}: status:`, status);
		// 	});
		// 	stream.on("metadata", (metadata: grpcWeb.Metadata) => {
		// 		console.log(`${debug}: metadata:`, metadata);
		// 	});
		// }
		stream.on("error", (err: grpcWeb.RpcError) => {
			console.error(`${debug}: unexpected stream error: code = ${err.code}` + `, message = "${err.message}"`);
			connectedWriter.set(false);
			restart();
		});
		stream.on("data", (response: DeviceInfo) => {
			// @ts-ignore
			console.log(`${debug}: data:`, response.toObject());
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
			if (debug !== "") console.log(`${debug}: done`);
			connectedWriter.set(false);
			restart();
		});
		if (debug !== "") console.log(`${debug}: started`);
	}

	runBackoff(debug, defaultMinBackoffMs, defaultMaxBackoffMs, run);

	return {
		subscribeData: dataWriter.subscribe,
		subscribeConnected: connectedWriter.subscribe,
	};
}

// // Run a function which should be restarted occasionally with some exponential
// // backoff.
// function runBackoff(debug: string, minBackoffMs: number, maxBackoffMs: number, run: (restart: VoidFunction) => void) {
// 	let running = false;
// 	let backoffDurationMs = 0;
// 	let lastBackoffTime = new Date();

// 	function doStart() {
// 		if (debug !== "") console.log(`${debug}: starting`);
// 		running = true;
// 		backoffDurationMs = 0;
// 		lastBackoffTime = new Date();
// 		run(doRestart);
// 	}

// 	function doRestart() {
// 		// Reset the backoff duration if the backoff has not been used for a
// 		// suitable amount of time.
// 		const now = new Date();
// 		const dt = differenceInMilliseconds(now, lastBackoffTime);
// 		if (dt > backoffDurationMs) {
// 			if (debug !== "") console.log(`${debug}: backoff reset after ${dt} ms`);
// 			backoffDurationMs = minBackoffMs;
// 		}
// 		lastBackoffTime = now;
// 		if (debug !== "") console.log(`${debug}: starting backoff for ${backoffDurationMs} ms`);
// 		setTimeout(() => {
// 			if (debug !== "") console.log(`${debug}: backoff finished`);
// 			backoffDurationMs = backoffDurationMs * 2
// 			if (backoffDurationMs > maxBackoffMs) {
// 				backoffDurationMs = maxBackoffMs;
// 			}
// 			if (running) {
// 				run(doRestart);
// 			}
// 		}, backoffDurationMs);
// 	}

// 	function doStop() {
// 		if (debug !== "") console.log(`${debug}: stopping`);
// 		running = false;
// 	}

// 	// run(doRestart);

// 	return {
// 		start: doStart,
// 		stop: doStop,
// 	}
// }

// function createDeviceInfosStream(debug: string) {
// 	let data: DeviceInfo[] = [];
// 	let connected: boolean = false;
// 	const dataWriter = writable(data);
// 	const connectedWriter = writable(connected);

// 	function run(restart: VoidFunction) {
// 		if (debug !== "") console.log(`${debug}: starting...`);
// 		const request = new GetDeviceInfosRequest();
// 		const stream = reactorClient.getDeviceInfos(request);
// 		// if (debug !== "") {
// 		// 	stream.on("status", (status: grpcWeb.Status) => {
// 		// 		console.log(`${debug}: status:`, status);
// 		// 	});
// 		// 	stream.on("metadata", (metadata: grpcWeb.Metadata) => {
// 		// 		console.log(`${debug}: metadata:`, metadata);
// 		// 	});
// 		// }
// 		stream.on("error", (err: grpcWeb.RpcError) => {
// 			console.error(`${debug}: unexpected stream error: code = ${err.code}` + `, message = "${err.message}"`);
// 			connectedWriter.set(false);
// 			restart();
// 		});
// 		stream.on("data", (response: DeviceInfo) => {
// 			// @ts-ignore
// 			console.log(`${debug}: data:`, response.toObject());
// 			connectedWriter.set(true);
// 			if (response.getBridgeId() !== "") {
// 				dataWriter.update(u => {
// 					const response_id = response.getBridgeId() + "." + response.getDeviceId();
// 					let updated = false;
// 					for (let i = 0; i < u.length; i++) {
// 						const u_id = u[i].getBridgeId() + "." + u[i].getDeviceId();
// 						if (u_id === response_id) {
// 							updated = true;
// 							u[i] = response;
// 							break;
// 						}
// 					}
// 					if (!updated) u = [...u, response];
// 					return u
// 				});
// 			}
// 		});
// 		stream.on("end", () => {
// 			if (debug !== "") console.log(`${debug}: done`);
// 			connectedWriter.set(false);
// 			restart();
// 		});
// 		if (debug !== "") console.log(`${debug}: started`);
// 	}

// 	const backoff = runBackoff(debug, defaultMinBackoffMs, defaultMaxBackoffMs, run);

// 	return {
// 		start: backoff.start,
// 		stop: backoff.stop,
// 		subscribeData: dataWriter.subscribe,
// 		subscribeConnected: connectedWriter.subscribe,
// 	};
// }

// function createDeviceInfosStream(debug: string) {
// 	let data: DeviceInfo[] = [];
// 	let connected: boolean = false;
// 	const dataWriter = writable(data);
// 	const connectedWriter = writable(connected);

// 	const minBackoffMs = 1000;
// 	const maxBackoffMs = 30000;
// 	let backoffDurationMs = 0;
// 	let lastBackoffTime = new Date();

// 	function start() {
// 		if (debug !== "") console.log(`${debug}: starting...`);
// 		const request = new GetDeviceInfosRequest();
// 		const stream = reactorClient.getDeviceInfos(request);
// 		// if (debug !== "") {
// 		// 	stream.on("status", (status: grpcWeb.Status) => {
// 		// 		console.log(`${debug}: status:`, status);
// 		// 	});
// 		// 	stream.on("metadata", (metadata: grpcWeb.Metadata) => {
// 		// 		console.log(`${debug}: metadata:`, metadata);
// 		// 	});
// 		// }
// 		stream.on("error", (err: grpcWeb.RpcError) => {
// 			console.error(`${debug}: unexpected stream error: code = ${err.code}` + `, message = "${err.message}"`);
// 			restart();
// 		});
// 		stream.on("data", (response: DeviceInfo) => {
// 			// @ts-ignore
// 			console.log(`${debug}: data:`, response.toObject());
// 			connectedWriter.set(true);
// 			if (response.getBridgeId() !== "") {
// 				dataWriter.update(u => {
// 					const response_id = response.getBridgeId() + "." + response.getDeviceId();
// 					let updated = false;
// 					for (let i = 0; i < u.length; i++) {
// 						const u_id = u[i].getBridgeId() + "." + u[i].getDeviceId();
// 						if (u_id === response_id) {
// 							updated = true;
// 							u[i] = response;
// 							break;
// 						}
// 					}
// 					if (!updated) u = [...u, response];
// 					return u
// 				});
// 			}
// 		});
// 		stream.on("end", () => {
// 			if (debug !== "") console.log(`${debug}: done`);
// 			restart();
// 		});
// 		if (debug !== "") console.log(`${debug}: started`);
// 	}

// 	function restart() {
// 		// Reset the backoff duration if the backoff has not been used for a
// 		// suitable amount of time.
// 		connectedWriter.set(false);
// 		const now = new Date();
// 		const dt = differenceInMilliseconds(now, lastBackoffTime);
// 		if (dt > backoffDurationMs) {
// 			if (debug !== "") console.log(`${debug}: backoff reset after ${dt} ms`);
// 			backoffDurationMs = minBackoffMs;
// 		}
// 		lastBackoffTime = now;
// 		if (debug !== "") console.log(`${debug}: starting backoff for ${backoffDurationMs} ms`);
// 		setTimeout(() => {
// 			if (debug !== "") console.log(`${debug}: backoff finished`);
// 			backoffDurationMs = backoffDurationMs * 2
// 			if (backoffDurationMs > maxBackoffMs) {
// 				backoffDurationMs = maxBackoffMs;
// 			}
// 			start();
// 		}, backoffDurationMs);
// 	}

// 	start();

// 	return {
// 		subscribeData: dataWriter.subscribe,
// 		subscribeConnected: connectedWriter.subscribe,
// 	};
// }

export const deviceInfosStream = createDeviceInfosStream("getDeviceInfos");
