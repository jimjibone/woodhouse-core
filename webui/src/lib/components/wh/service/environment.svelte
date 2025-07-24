<script lang="ts">
	import type { Attribute, FloatAttribute } from '$lib/api/v1/clients/client_service_pb';
	import ServiceRoot, { type StandardProps } from "./service-root.svelte";
	import ServiceAction from './service-action.svelte';
	import { ThermometerIcon } from '@lucide/svelte';
	import { OthersContent } from '$lib/components/wh/attributes';

	let {
		deviceID,
		service,
		...rest
	}: StandardProps = $props();

	let attrTemperature: FloatAttribute | undefined = $state(undefined);
	let attrHumidity: FloatAttribute | undefined = $state(undefined);
	let attrPressure: FloatAttribute | undefined = $state(undefined);
	let attrOthers: Attribute[] = $state([]);

	$effect(() => {
		let others: Attribute[] = [];
		for (const attr of service.attrs) {
			if (attr.id === 'temperature') {
				attrTemperature = attr.float;
			} else if (attr.id === 'humidity') {
				attrHumidity = attr.float;
			} else if (attr.id === 'pressure') {
				attrPressure = attr.float;
			} else {
				others = [...others, attr];
			}
		}
		attrOthers = others;
	});

	let serviceAction = new ServiceAction(deviceID, service.id);
</script>

{#snippet icon()}
	<ThermometerIcon/>
{/snippet}

{#snippet details()}
	{#if attrTemperature !== undefined}
		<p>
			{attrTemperature.value.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })}°C
		</p>
	{/if}
	{#if attrHumidity !== undefined}
		<!-- <ThermometerIcon class="text-muted-foreground size-5"/> -->
		<p class="text-muted-foreground">
			{attrHumidity.value.toLocaleString(undefined, { maximumFractionDigits: 0 })}%RH
		</p>
	{/if}
	{#if attrPressure !== undefined}
		<!-- <PowerIcon class="text-muted-foreground size-5"/> -->
		<p class="text-muted-foreground">
			{attrPressure.value.toLocaleString(undefined, { maximumFractionDigits: 0 })}hPa
		</p>
	{/if}
{/snippet}

<ServiceRoot
	{deviceID}
	{...rest}
	service={service}
	icon={icon}
	details={details}
>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if attrTemperature !== undefined}
			<div>Temperature</div>
			<div class="col-span-2">
				{attrTemperature.value.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })}°C
			</div>
		{/if}
		{#if attrHumidity !== undefined}
			<div>Humidity</div>
			<div class="col-span-2">
				{attrHumidity.value.toLocaleString(undefined, { maximumFractionDigits: 0 })}%RH
			</div>
		{/if}
		{#if attrPressure !== undefined}
			<div>Pressure</div>
			<div class="col-span-2">
				{attrPressure.value.toLocaleString(undefined, { maximumFractionDigits: 0 })}hPa
			</div>
		{/if}
		<OthersContent others={attrOthers} {serviceAction}/>
	</div>
</ServiceRoot>
