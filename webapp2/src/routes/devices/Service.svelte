<script lang="ts">
	import { Service, Service_ServiceType, Value } from '$lib/api/v1/clients/client_service_pb';
	// import { getDeviceInfo, getDeviceName } from '$lib/apitools';
	import Attribute from './Attribute.svelte';

	import * as Card from '$lib/components/ui/card';
	import Button from '@/components/ui/button/button.svelte';

	export let online: boolean;
	export let service: Service;
	export let onAction: (serviceID: string, val: Value) => Promise<void> | undefined

	let action = async (val: Value) => {
		if (onAction) {
			onAction(service.id, val);
		}
	}

	// $:info = getDeviceInfo(device);
</script>

{#if !(service.typ === Service_ServiceType.INFO || service.typ === Service_ServiceType.ONLINE)}
<Card.Root class={online ? "" : "bg-muted"}>
	<Card.Header class="pb-3">
		<Card.Title>{service.id}: {service.alias}: {Service_ServiceType[service.typ]}</Card.Title>
	</Card.Header>
	<Card.Content class="grid grid-cols-1 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-4">
		{#each service.attrs as attr, i}
		<Attribute online={online} attr={attr} onAction={action}/>
		{/each}
	</Card.Content>
	<Card.Footer>
	</Card.Footer>
</Card.Root>
{/if}
