<script lang="ts">
	import { Device, Device_DeviceType, Value } from '$lib/api/v1/clients/client_service_pb';
	import { getDeviceInfo, getDeviceName } from '$lib/apitools';

	import * as Card from '$lib/components/ui/card';
	import Button from '@/components/ui/button/button.svelte';
	import Service from './Service.svelte';

	export let device: Device;
	export let onAction: (
		deviceID: string,
		serviceID: string,
		vals: Value[]
	) => Promise<void> | undefined;

	let action = async (serviceID: string, vals: Value[]) => {
		if (onAction) {
			onAction(device.id, serviceID, vals);
		}
	};

	$: info = getDeviceInfo(device);
</script>

<Card.Root class={info.online ? '' : 'bg-muted'}>
	<Card.Header class="pb-3">
		<Card.Title>{info.name}</Card.Title>
	</Card.Header>
	<Card.Content>
		<div class="flex flex-col gap-2">
			<p class="max-sm:hidden">
				{device.id}, {Device_DeviceType[device.typ]}, {info.online
					? 'online'
					: 'offline'}{info.web_url !== '' ? ', ' + info.web_url : ''}
			</p>
			{#each device.services as srv, i (srv.id)}
				<Service deviceID={device.id} online={info.online} service={srv} onAction={action} />
			{/each}
		</div>
	</Card.Content>
	<Card.Footer>
		<Button href="/devices/{device.id}">Open</Button>
	</Card.Footer>
</Card.Root>
