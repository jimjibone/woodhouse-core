import { writable } from 'svelte/store';
import type * as grpcWeb from 'grpc-web';
import type { BridgeInfo } from '../api/bridge_pb';
import { ReactorServiceClient } from '../api/Reactor_serviceServiceClientPb';
import { GetBridgeInfosRequest } from '../api/reactor_service_pb';
import { createBackoff, defaultMinBackoffMs, defaultMaxBackoffMs } from './utils';

const reactorClient = new ReactorServiceClient('/api');

export const bridgeInfosStream = createBridgeInfosStream("getBridgeInfos", true);
function createBridgeInfosStream(name: string, debug: boolean) {
	let data: BridgeInfo[] = [];
	let connected: boolean = false;
	let stream: grpcWeb.ClientReadableStream<BridgeInfo> = null;
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
		const request = new GetBridgeInfosRequest();
		stream = reactorClient.getBridgeInfos(request);
		stream.on("error", (err: grpcWeb.RpcError) => {
			console.error(`${name}: unexpected stream error: code = ${err.code}` + `, message = "${err.message}"`);
			connectedWriter.set(false);
			restart();
		});
		stream.on("data", (response: BridgeInfo) => {
			if (debug) console.log(`${name}: data:`, response.toObject());
			connectedWriter.set(true);
			if (response.getBridgeId() !== "") {
				dataWriter.update(u => {
					let updated = false;
					for (let i = 0; i < u.length; i++) {
						if (u[i].getBridgeId() === response.getBridgeId()) {
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
