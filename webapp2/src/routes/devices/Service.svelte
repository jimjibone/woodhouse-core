<script lang="ts">
	import { Service, Service_ServiceType, Value } from '$lib/api/v1/clients/client_service_pb';
	import Attribute from './Attribute.svelte';

	import * as Card from '$lib/components/ui/card';
	import { Label } from "$lib/components/ui/label/index.js";
	import { ScrollArea } from "$lib/components/ui/scroll-area/index.js";

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
<div class={online ? "" : "bg-muted"}>
	<div class="pb-3">
		<Label class="max-sm:hidden">{service.id}: {service.alias}: {Service_ServiceType[service.typ]}</Label>
		<Label class="sm:hidden">{service.id}</Label>
	</div>
	<div>
		<ScrollArea class="w-auto whitespace-nowrap" orientation="horizontal">
			<div class="flex w-auto space-x-4">
				{#each service.attrs as attr, i}
				<Attribute online={online} attr={attr} onAction={action}/>
				{/each}
			</div>
		</ScrollArea>
	</div>
</div>
{/if}
