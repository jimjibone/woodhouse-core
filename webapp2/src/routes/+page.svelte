<script lang="ts">
	import { createPromiseClient, ConnectError } from '@connectrpc/connect';
	import type { Transport } from '@connectrpc/connect';
	import { UserService } from '$lib/api/v1/clients/user_service_connect';
	import { DevicesStreamRequest } from '$lib/api/v1/clients/user_service_pb';
	import { ActionRequest, Value, type ActionResponse, BoolValue, Device, Service, Attribute, Service_ServiceType, Device_DeviceType } from '$lib/api/v1/clients/client_service_pb';
	// import { ElizaService } from "../gen/connectrpc/eliza/v1/eliza_connect.js";
	// import { IntroduceRequest } from "../gen/connectrpc/eliza/v1/eliza_pb.js";
	import { getContext } from 'svelte';
	import Button from '@/components/ui/button/button.svelte';
	import * as Card from "$lib/components/ui/card";

	let devices: Device[] = [];
	let responses: ActionResponse[] = [];

	const transport: Transport = getContext('transport');

	// Make the Eliza Service client
	const client = createPromiseClient(UserService, transport);

	const updateDevice = (prev: Device, next: Device): Device => {
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

	const getDeviceName = (dev: Device): string => {
		for (const ser of dev.services) {
			if (ser.typ === Service_ServiceType.INFO) {
				for (const attr of ser.attrs) {
					if (attr.id === "name") {
						return attr.text!.value;
					}
				}
			}
		}
		return dev.id;
	}

	const streamDevices = async () => {
		const request = new DevicesStreamRequest({});
		for await (const response of client.devicesStream(request)) {
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
	};
	streamDevices();

	const sendOn = async (deviceId: string) => {
		await send(deviceId, true)
	}
	const sendOff = async (deviceId: string) => {
		await send(deviceId, false)
	}
	const send = async (deviceId: string, val: boolean) => {
		const request = new ActionRequest({
			// Extension Dimmer
			deviceId: deviceId, //"shellydimmer2-DEADBEEF",
			serviceId: "lightbulb",
			values: [
				new Value({
					id: "on",
					bool: new BoolValue({
						value: val
					})
				})
			]
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

<div class="relative flex gap-4 h-full min-h-[50vh] flex-col rounded-xl lg:col-span-3">
{#each devices as dev, i}
<Card.Root class="">
	<Card.Header class="pb-3">
		<Card.Title>{getDeviceName(dev)}</Card.Title>
	</Card.Header>
	<Card.Content>
		<p>{dev.id}, {Device_DeviceType[dev.typ]}</p>
	</Card.Content>
	<Card.Footer>
		<Button on:click={() => sendOn(dev.id)}>On</Button>
		<Button on:click={() => sendOff(dev.id)}>Off</Button>
	</Card.Footer>
</Card.Root>
{/each}


<!-- {#each devices as dev, i}
<div>
	<p>{dev.toJsonString()}</p>
</div>
{/each} -->
</div>
