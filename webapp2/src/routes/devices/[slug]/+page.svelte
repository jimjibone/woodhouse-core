<script lang="ts">
	import { createPromiseClient, ConnectError } from '@connectrpc/connect';
	import type { Transport } from '@connectrpc/connect';
	import { getContext } from 'svelte';
	import { page } from '$app/stores';
	import { UserService } from '@/api/v1/clients/user_service_connect';
	import { ActionRequest, Device, Value, type Service, BoolValue, ActionResponse } from '@/api/v1/clients/client_service_pb';
	import { DevicesStreamRequest } from '@/api/v1/clients/user_service_pb';
	import DeviceComponent from '../Device.svelte';

	const deviceId = $page.params.slug;

	let device: Device | undefined;
	let responses: ActionResponse[] = [];

	const transport: Transport = getContext('transport');

	// Make the User Service client
	const client = createPromiseClient(UserService, transport);

	const updateDevice = (prev: Device|undefined, next: Device): Device => {
		if (prev === undefined) {
			return next;
		}
		prev.typ = next.typ;
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

	const streamDevices = async () => {
		const request = new DevicesStreamRequest({
			includeDeviceIds: [
				deviceId
			]
		});
		for await (const response of client.devicesStream(request)) {
			device = updateDevice(device, response);
		}
	};
	streamDevices();

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
	<h1 class="text-xl font-semibold">Device: {deviceId}</h1>
</header>
<main class="grid flex-1 gap-4 overflow-auto p-4">
	<div class="relative flex gap-4 h-full min-h-[50vh] flex-col rounded-xl">
	{#if device !== undefined}
		<DeviceComponent device={device} onAction={action} />
	{:else}
		<p>No device found</p>
	{/if}
</main>
