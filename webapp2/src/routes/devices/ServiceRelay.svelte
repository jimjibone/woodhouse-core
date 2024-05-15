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
	let attrVoltage: AttributeType | undefined
	let attrCurrent: AttributeType | undefined
	let attrPower: AttributeType | undefined
	let attrTemperature: AttributeType | undefined
	let attrOthers: AttributeType[]

	$:{
		attrOthers = [];
		for (const attr of service.attrs) {
			if (attr.id === "on") {
				attrOn = attr;
			} else if (attr.id === "voltage") {
				attrVoltage = attr;
			} else if (attr.id === "current") {
				attrCurrent = attr;
			} else if (attr.id === "power") {
				attrPower = attr;
			} else if (attr.id === "temperature") {
				attrTemperature = attr;
			} else {
				attrOthers = [...attrOthers, attr];
			}
		}
	}
</script>

{#if service.typ === Service_ServiceType.RELAY}
<div class={online ? "" : "bg-muted"}>
	<div class="pb-3">
		<ServiceHeader id={service.id} alias={service.alias}/>
	</div>
	<div>
		<div class="flex w-auto space-x-4">
			{#if attrOn !== undefined}
			<Attribute online={online} attr={attrOn} onAction={action}/>
			{/if}
			{#if attrVoltage !== undefined}
			<Attribute online={online} attr={attrVoltage} onAction={action}/>
			{/if}
			{#if attrCurrent !== undefined}
			<Attribute online={online} attr={attrCurrent} onAction={action}/>
			{/if}
			{#if attrPower !== undefined}
			<Attribute online={online} attr={attrPower} onAction={action}/>
			{/if}
			{#if attrTemperature !== undefined}
			<Attribute online={online} attr={attrTemperature} onAction={action}/>
			{/if}
			{#each attrOthers as attr, i}
			<Attribute online={online} attr={attr} onAction={action}/>
			{/each}
		</div>
	</div>
</div>
{:else}
<p>ERROR Service Type {Service_ServiceType[service.typ]} is not RELAY</p>
{/if}
