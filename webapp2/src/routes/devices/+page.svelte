<script lang="ts">
	import { createPromiseClient, createCallbackClient, ConnectError, Code } from '@connectrpc/connect';
	import type { Transport } from '@connectrpc/connect';
	import { UserService } from '$lib/api/v1/clients/user_service_connect';
	import { DevicesStreamRequest } from '$lib/api/v1/clients/user_service_pb';
	import { ActionRequest, Value, type ActionResponse, BoolValue, Device, Service, Attribute, Service_ServiceType, Device_DeviceType } from '$lib/api/v1/clients/client_service_pb';
	import { getContext } from 'svelte';
	import DeviceComponent from './Device.svelte';
	import { getDeviceName } from '$lib/apitools';

	let connected = true;
	let backoff = 1000;
	const minBackoff = 1000;
	const maxBackoff = 16000;
	let devices: Device[] = [];
	let responses: ActionResponse[] = [];

	const transport: Transport = getContext('transport');

	// Make the User Service client
	const client = createPromiseClient(UserService, transport);

	const updateDevice = (prev: Device, next: Device): Device => {
		prev.typ = next.typ;
		if (next.fullState) {
			// Remove all services as we're about to receive the complete new set.
			prev.services = [];
		}
		for (let i = 0; i < next.services.length; i++) {
			let foundService = false;
			for (let j = 0; j < prev.services.length; j++) {
				if (next.services[i].id === prev.services[j].id) {
					foundService = true;
					prev.services[j] = updateService(prev.services[j], next.services[i]);
					break;
				}
			}
			if (!foundService) {
				if (!foundService) prev.services = [...prev.services, next.services[i]];
			}
		}
		return prev;
	}

	const updateService = (prev: Service, next: Service): Service => {
		prev.typ = next.typ;
		prev.alias = next.alias;
		for (let i = 0; i < next.attrs.length; i++) {
			let foundAttr = false;
			for (let j = 0; j < prev.attrs.length; j++) {
				if (next.attrs[i].id === prev.attrs[j].id) {
					foundAttr = true;
					prev.attrs[j] = next.attrs[i];
					break;
				}
			}
			if (!foundAttr) {
				if (!foundAttr) prev.attrs = [...prev.attrs, next.attrs[i]];
			}
		}
		return prev;
	}

	const streamDevices = async (onFinish: (resetBackoff: boolean) => void) => {
		// Use a timer to periodically check if the connection has died. If it
		// has then trigger the abort controller to cancel the stream and then
		// fire up another one after a backoff delay.
		let lastrx = Date.now();
		const controller = new AbortController();
		const interval = setInterval(() => {
			const now = Date.now();
			if (now - lastrx > 11000) {
				controller.abort();
			}
		}, 1000);
		const request = new DevicesStreamRequest({});
		try {
			for await (const response of client.devicesStream(request, { signal: controller.signal })) {
				// ID will be empty if this is a keepalive message.
				if (response.id !== "") {
					// console.log("device: " + response.toJsonString());
					let foundDevice = false;
					for (let d = 0; d < devices.length; d++) {
						if (devices[d].id === response.id) {
							foundDevice = true;
							devices[d] = updateDevice(devices[d], response);
							break;
						}
					}
					if (!foundDevice) devices = [...devices, response];

					devices = devices.sort((a, b) => {
						const aName = getDeviceName(a);
						const bName = getDeviceName(b);
						return aName > bName ? 1 : (bName > aName ? -1 : 0);
					})
				}
				lastrx = Date.now();
				connected = true;
			}
		} catch (err) {
			if (err instanceof ConnectError) {
				if (err.code !== Code.Unknown) {
					console.error("error stream: (" + err.code + ") " + err.message);
				}
			}
			clearInterval(interval);
			const resetBackoff = connected;
			connected = false;
			onFinish(resetBackoff);
		}
	};

	const retryStreamDevices = async () => {
		streamDevices((resetBackoff: boolean) => {
			if (resetBackoff) {
				backoff = minBackoff;
			} else {
				backoff = backoff * 2;
				if (backoff > maxBackoff) {
					backoff = maxBackoff;
				}
			}
			setTimeout(() => {
				retryStreamDevices();
			}, backoff);
		});
	};
	retryStreamDevices();

	const action = async (deviceID: string, serviceID: string, val: Value) => {
		const request = new ActionRequest({
			deviceId: deviceID,
			serviceId: serviceID,
			values: [ val ]
		});
		console.log("sending action: " + request.toJsonString());
		try {
			for await (const response of client.sendAction(request)) {
				console.log("received action: " + response.toJsonString());
				responses = [...responses, response];
			}
		} catch (err) {
			if (err instanceof ConnectError) {
				console.error("error action: " + err.message);
			}
		}
	};
</script>

<header class="bg-background sticky top-0 z-10 flex h-[57px] items-center gap-1 border-b px-4">
	<h1 class="text-xl font-semibold">Devices{connected ? "" : " - Disconnected (backoff=" + backoff + "ms)"}</h1>
</header>
<main class="grid flex-1 gap-4 overflow-auto p-4">
	<div class="relative flex gap-4 h-full min-h-[50vh] flex-col rounded-xl">
		{#each devices as dev, i}
		<DeviceComponent device={dev} onAction={action}/>
		{/each}
	</div>
</main>
