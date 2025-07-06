<script lang="ts">
	import type { Attribute, FloatAttribute, IntAttribute, Service } from '$lib/api/v1/clients/client_service_pb';
	import ServiceRoot from "./service-root.svelte";
	import ServiceAction from './service-action.svelte';
	import { BatteryFullIcon, BatteryLowIcon, BatteryMediumIcon, BatteryWarningIcon } from '@lucide/svelte';
	import { OthersContent } from '$lib/components/wh/attributes';

	let {
		deviceName,
		showDeviceName,
		deviceID,
		online,
		service
	}: {
		deviceName: string,
		showDeviceName?: boolean,
		deviceID: string,
		online: boolean,
		service: Service
	} = $props();

	let attrLevel: IntAttribute | undefined = $state(undefined);
	let attrVoltage: FloatAttribute | undefined = $state(undefined);
	let attrOthers: Attribute[] = $state([]);
	let level: number = $state(100);

	$effect(() => {
		let others: Attribute[] = [];
		for (const attr of service.attrs) {
			if (attr.id === 'level') {
				attrLevel = attr.int;
				level = Number(attr.int?.value);
			} else if (attr.id === 'voltage') {
				attrVoltage = attr.float;
			} else {
				others = [...others, attr];
			}
		}
		attrOthers = others;
	});

	let serviceAction = new ServiceAction(deviceID, service.id);
</script>

{#snippet icon()}
	{#if level < 20}
		<BatteryWarningIcon/>
	{:else if level < 33}
		<BatteryLowIcon/>
	{:else if level < 66}
		<BatteryMediumIcon/>
	{:else}
		<BatteryFullIcon/>
	{/if}
{/snippet}

{#snippet details()}
	{#if attrLevel !== undefined}
		<!-- <PowerIcon class="text-muted-foreground size-5"/> -->
		<p>
			{attrLevel.value.toLocaleString(undefined, { maximumFractionDigits: 0 })}%
		</p>
	{/if}
	{#if attrVoltage !== undefined}
		<!-- <ThermometerIcon class="text-muted-foreground size-5"/> -->
		<p class="text-muted-foreground">
			{attrVoltage.value.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })}V
		</p>
	{/if}
{/snippet}

<ServiceRoot
	deviceName={deviceName}
	showDeviceName={showDeviceName}
	deviceID={deviceID}
	online={online}
	service={service}
	icon={icon}
	iconclass={level < 10 ? "bg-red-400 text-black" : level < 20 ? "bg-yellow-400 text-black" : false}
	details={details}
>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if attrLevel !== undefined}
			<div>Level</div>
			<div class="col-span-2">
				{attrLevel.value.toLocaleString(undefined, { maximumFractionDigits: 0 })}%
			</div>
		{/if}
		{#if attrVoltage !== undefined}
			<div>Current</div>
			<div class="col-span-2">
				{attrVoltage.value.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })}V
			</div>
		{/if}
		<OthersContent others={attrOthers} {serviceAction}/>
	</div>
</ServiceRoot>
