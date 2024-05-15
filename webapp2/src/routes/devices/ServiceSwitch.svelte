<script lang="ts">
	import { Service, Service_ServiceType, Value, Attribute as AttributeType } from '$lib/api/v1/clients/client_service_pb';
	import Attribute from './Attribute.svelte';
	import ServiceHeader from './ServiceHeader.svelte';

	export let online: boolean;
	export let service: Service;
	export let onAction: (serviceID: string, val: Value) => Promise<void> | undefined

	let action = async (val: Value) => {
		if (onAction) {
			onAction(service.id, val);
		}
	}

	let attrOn: AttributeType | undefined
	let attrOthers: AttributeType[]

	$:{
		attrOthers = [];
		for (const attr of service.attrs) {
			if (attr.id === "on") {
				attrOn = attr;
			} else {
				attrOthers = [...attrOthers, attr];
			}
		}
	}
</script>

{#if service.typ === Service_ServiceType.SWITCH}
<div class={online ? "" : "bg-muted"}>
	<div class="pb-3">
		<ServiceHeader id={service.id} alias={service.alias}/>
	</div>
	<div>
		<div class="flex w-auto space-x-4">
			{#if attrOn !== undefined}
			<Attribute online={online} attr={attrOn} onAction={action}/>
			{/if}
			{#each attrOthers as attr, i}
			<Attribute online={online} attr={attr} onAction={action}/>
			{/each}
		</div>
	</div>
</div>
{:else}
<p>ERROR Service Type {Service_ServiceType[service.typ]} is not SWITCH</p>
{/if}
