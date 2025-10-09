<script lang="ts">
	import { ValueSchema, BoolValueSchema } from '$lib/api/v1/clients/client_service_pb';
	import type { Attribute, BoolAttribute, FloatAttribute } from '$lib/api/v1/clients/client_service_pb';
	import ServiceRoot, { type StandardProps } from "./service-root.svelte";
	import ServiceAction from './service-action.svelte';
	import { PowerIcon } from '@lucide/svelte';
	import { create } from '@bufbuild/protobuf';
	import { BoolContent, OthersContent } from '$lib/components/wh/attributes';

	let {
		deviceID,
		service,
		...rest
	}: StandardProps = $props();

	let attrOn: BoolAttribute | undefined = $state(undefined);
	let attrVoltage: FloatAttribute | undefined = $state(undefined);
	let attrCurrent: FloatAttribute | undefined = $state(undefined);
	let attrPower: FloatAttribute | undefined = $state(undefined);
	let attrTemperature: FloatAttribute | undefined = $state(undefined);
	let attrOthers: Attribute[] = $state([]);
	let on: boolean = $state(false);

	$effect(() => {
		let others: Attribute[] = [];
		for (const attr of service.attrs) {
			if (attr.id === 'on') {
				attrOn = attr.bool;
				on = attr.bool?.value!;
			} else if (attr.id === 'voltage') {
				attrVoltage = attr.float;
			} else if (attr.id === 'current') {
				attrCurrent = attr.float;
			} else if (attr.id === 'power') {
				attrPower = attr.float;
			} else if (attr.id === 'temperature') {
				attrTemperature = attr.float;
			} else {
				others = [...others, attr];
			}
		}
		attrOthers = others;
	});

	let serviceAction = new ServiceAction(deviceID, service.id);

	let sendActionOn = async (val: boolean) => {
		serviceAction.send([
			create(ValueSchema, {
				id: 'on',
				bool: create(BoolValueSchema, {
					value: val
				})
			})
		]);
	};
</script>

{#snippet icon()}
	<PowerIcon/>
{/snippet}

{#snippet details()}
	{#if attrOn !== undefined}
		<p>{attrOn.value ? 'On' : 'Off'}</p>
	{/if}
	{#if attrPower !== undefined}
		<p class="text-muted-foreground">
			{attrPower.value.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })} W
		</p>
	{:else if attrCurrent !== undefined}
		<p class="text-muted-foreground">
			{attrCurrent.value.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })} A
		</p>
	{:else if attrVoltage !== undefined}
		<p class="text-muted-foreground">
			{attrVoltage.value.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })} V
		</p>
	{/if}
{/snippet}

<ServiceRoot
	{deviceID}
	{...rest}
	service={service}
	icon={icon}
	iconclass={on ? "bg-green-400 text-black" : false}
	details={details}
>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if attrOn !== undefined}
			<BoolContent
				name="On"
				attr={attrOn}
				onaction={sendActionOn}
			/>
		{/if}
		{#if attrVoltage !== undefined}
			<div>Voltage</div>
			<div class="col-span-2">
				{attrVoltage.value.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })} V
			</div>
		{/if}
		{#if attrCurrent !== undefined}
			<div>Current</div>
			<div class="col-span-2">
				{attrCurrent.value.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })} A
			</div>
		{/if}
		{#if attrPower !== undefined}
			<div>Power</div>
			<div class="col-span-2">
				{attrPower.value.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })} W
			</div>
		{/if}
		{#if attrTemperature !== undefined}
			<div>Temperature</div>
			<div class="col-span-2">
				{attrTemperature.value.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })}°C
			</div>
		{/if}
		<OthersContent others={attrOthers} {serviceAction}/>
	</div>
</ServiceRoot>
