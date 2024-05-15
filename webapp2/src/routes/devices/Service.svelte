<script lang="ts">
	import { Service, Service_ServiceType, Value } from '$lib/api/v1/clients/client_service_pb';

	import { ScrollArea } from "$lib/components/ui/scroll-area/index.js";
	import ServiceHeader from './ServiceHeader.svelte';
	import ServiceRelay from './ServiceRelay.svelte';
	import ServiceSwitch from './ServiceSwitch.svelte';
	import BoxedAttribute from './BoxedAttribute.svelte';

	export let online: boolean;
	export let service: Service;
	export let onAction: (serviceID: string, val: Value) => Promise<void> | undefined

	let action = async (val: Value) => {
		if (onAction) {
			onAction(service.id, val);
		}
	}
</script>

{#if !(service.typ === Service_ServiceType.INFO || service.typ === Service_ServiceType.ONLINE)}
	{#if service.typ === Service_ServiceType.RELAY}
	<ServiceRelay online={online} service={service} onAction={onAction} />
	{:else if service.typ === Service_ServiceType.SWITCH}
	<ServiceSwitch online={online} service={service} onAction={onAction} />
	{:else}
	<div class={online ? "" : "bg-muted"}>
		<div class="pb-3">
			<ServiceHeader id={service.id} alias={service.alias}/>
		</div>
		<div>
			<ScrollArea class="w-auto whitespace-nowrap" orientation="horizontal">
				<div class="flex w-auto space-x-4">
					{#each service.attrs as attr, i}
					<BoxedAttribute online={online} attr={attr} onAction={action}/>
					{/each}
				</div>
			</ScrollArea>
		</div>
	</div>
	{/if}
{/if}
