<script lang="ts">
	import { Device, Device_DeviceType, Value } from '$lib/api/v1/clients/client_service_pb';
	import { getDeviceInfo, getDeviceName } from '$lib/apitools';

	import * as Card from '$lib/components/ui/card';
	import Button from '@/components/ui/button/button.svelte';
	import Service from './Service.svelte';

	export let device: Device;
	export let onAction: (deviceID: string, serviceID: string, val: Value) => Promise<void> | undefined

	let action = async (serviceID: string, val: Value) => {
		if (onAction) {
			onAction(device.id, serviceID, val);
		}
	}

	$:info = getDeviceInfo(device);
</script>

<Card.Root class={info.online ? "" : "bg-muted"}>
	<Card.Header class="pb-3">
		<Card.Title>{info.name}</Card.Title>
	</Card.Header>
	<Card.Content>
		<p class="max-sm:hidden">{device.id}, {Device_DeviceType[device.typ]}, {info.online ? "online" : "offline"}</p>
		{#each device.services as srv, i}
		<Service online={info.online} service={srv} onAction={action}/>
		{/each}
	</Card.Content>
	<Card.Footer>
		<Button href="/devices/{device.id}">Open</Button>
	</Card.Footer>
</Card.Root>
