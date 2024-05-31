<script lang="ts">
	import { Service, Service_ServiceType, Value, BoolValue, Attribute as AttributeType, BoolAttribute, FloatAttribute } from '$lib/api/v1/clients/client_service_pb';
	import { Power } from 'lucide-svelte';
	import { cn } from "$lib/utils.js";
	import { validators } from 'tailwind-merge';

	export let title: string | undefined = undefined;
	export let online: boolean;
	export let service: Service;
	export let onAction: ((serviceID: string, val: Value) => Promise<void>) | undefined

	let alias: string = (title ? title + (service.alias !== "" ? ": "+service.alias : "") : service.alias);
	let attrOn: BoolAttribute | undefined
	let attrVoltage: FloatAttribute | undefined
	let attrCurrent: FloatAttribute | undefined
	let attrPower: FloatAttribute | undefined
	let attrTemperature: FloatAttribute | undefined
	let attrOthers: AttributeType[]

	$:{
		attrOthers = [];
		for (const attr of service.attrs) {
			if (attr.id === "on") {
				attrOn = attr.bool;
			} else if (attr.id === "voltage") {
				attrVoltage = attr.float;
			} else if (attr.id === "current") {
				attrCurrent = attr.float;
			} else if (attr.id === "power") {
				attrPower = attr.float;
			} else if (attr.id === "temperature") {
				attrTemperature = attr.float;
			} else {
				attrOthers = [...attrOthers, attr];
			}
		}
	}

	let action = async (val: Value) => {
		if (onAction) {
			onAction(service.id, val);
		}
	}

	let actionOn = async (val: boolean) => {
		action(
			new Value({
				id: "on",
				bool: new BoolValue({
					value: val
				})
			})
		);
	}

	let actionOnToggle = async () => {
		if (attrOn !== undefined) {
			actionOn(!attrOn.value);
		}
	}
</script>

{#if service.typ === Service_ServiceType.RELAY}
<!-- <div class="grid grid-cols-2 gap-4"> -->
<div class={cn("p-2 rounded-lg border bg-card text-card-foreground shadow-sm", !online && "bg-muted")}>
	<div class="flex flex-row gap-2">
		<div class="shrink">
			<div class="h-full grid place-content-center">
				<button class={cn("p-2 rounded-full", attrOn?.value ? "bg-yellow-400 text-black" : "bg-secondary text-secondary-foreground")} on:click={actionOnToggle}>
					<Power/>
				</button>
			</div>
		</div>
		<div class="grow">
			<div class="h-full flex flex-col gap-0 justify-center">
				{#if alias !== ""}
				<div class="p-0 rounded-lg">
					<p class="font-semibold">{alias}</p>
				</div>
				{/if}
				<div class="p-0 rounded-lg flex flex-row gap-2">
					{#if attrOn !== undefined}
					<p>{attrOn.value ? "On" : "Off"}</p>
					{/if}
					{#if attrVoltage !== undefined}
					<p class="text-muted-foreground">{(attrVoltage.value).toLocaleString(undefined, { maximumFractionDigits: 0 })}V</p>
					{/if}
					{#if attrPower !== undefined}
					<p class="text-muted-foreground">{(attrPower.value).toLocaleString(undefined, { maximumFractionDigits: 0 })}W</p>
					{/if}
					{#if attrTemperature !== undefined}
					<p class="text-muted-foreground">{(attrTemperature.value).toLocaleString(undefined, { maximumFractionDigits: 0 })}°C</p>
					{/if}
				</div>
			</div>
		</div>
	</div>
</div>
<!-- <div class="p-4 rounded-lg shadow-lg bg-fuchsia-500">02</div>
<div class="p-4 rounded-lg shadow-lg bg-fuchsia-500">03</div>
</div> -->
{:else}
<p>ERROR Service Type {Service_ServiceType[service.typ]} is not RELAY</p>
{/if}
