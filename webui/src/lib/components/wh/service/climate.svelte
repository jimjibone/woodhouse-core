<script lang="ts">
	import { ValueSchema, FloatValueSchema } from '$lib/api/v1/clients/client_service_pb';
	import type { Attribute, FloatAttribute, IntAttribute, Service } from '$lib/api/v1/clients/client_service_pb';
	import ServiceRoot, { type StandardProps } from "./service-root.svelte";
	import ServiceAction from './service-action.svelte';
	import { FlameIcon } from '@lucide/svelte';
	import { create } from '@bufbuild/protobuf';
	import { FloatContent, OthersContent } from '$lib/components/wh/attributes';

	let {
		deviceID,
		service,
		...rest
	}: StandardProps = $props();

	let attrHeatingSetpoint: FloatAttribute | undefined = $state(undefined);
	let attrLocalTemperature: FloatAttribute | undefined = $state(undefined);
	let attrPIHeatingDemand: IntAttribute | undefined = $state(undefined);
	let attrOthers: Attribute[] = $state([]);

	$effect(() => {
		let others: Attribute[] = [];
		for (const attr of service.attrs) {
			if (attr.id === 'heating_setpoint') {
				attrHeatingSetpoint = attr.float;
			} else if (attr.id === 'local_temperature') {
				attrLocalTemperature = attr.float;
			} else if (attr.id === 'pi_heating_demand') {
				attrPIHeatingDemand = attr.int;
			} else {
				others = [...others, attr];
			}
		}
		attrOthers = others;
	});

	let serviceAction = new ServiceAction(deviceID, service.id);

	let sendActionHeatingSetpoint = async (val: number) => {
		serviceAction.send([
			create(ValueSchema, {
				id: 'heating_setpoint',
				float: create(FloatValueSchema, {
					value: val
				})
			})
		]);
	};
</script>

{#snippet icon()}
	<FlameIcon/>
{/snippet}

{#snippet details()}
	{#if attrHeatingSetpoint !== undefined}
		<!-- <GaugeIcon class="size-5"/> -->
		<p>
			{attrHeatingSetpoint.value.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })}°C
		</p>
	{/if}
	{#if attrLocalTemperature !== undefined}
		<!-- <ThermometerIcon class="text-muted-foreground size-5"/> -->
		<p class="text-muted-foreground">
			{attrLocalTemperature.value.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })}°C
		</p>
	{/if}
	{#if attrPIHeatingDemand !== undefined}
		<!-- <PowerIcon class="text-muted-foreground size-5"/> -->
		<p class="text-muted-foreground">
			{attrPIHeatingDemand.value.toLocaleString(undefined, { maximumFractionDigits: 0 })}%
		</p>
	{/if}
{/snippet}

<ServiceRoot
	{deviceID}
	{...rest}
	service={service}
	actionPending={serviceAction.pending}
	errorSignal={serviceAction.error}
	icon={icon}
	details={details}
>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if attrHeatingSetpoint !== undefined}
			<FloatContent
				name="Target"
				value={attrHeatingSetpoint.value}
				min={attrHeatingSetpoint.min}
				max={attrHeatingSetpoint.max}
				step={attrHeatingSetpoint.step}
				onaction={sendActionHeatingSetpoint}
				units="°C"
			/>
		{/if}
		{#if attrLocalTemperature !== undefined}
			<div>Current</div>
			<div class="col-span-2">
				{attrLocalTemperature.value.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })}°C
			</div>
		{/if}
		{#if attrPIHeatingDemand !== undefined}
			<div>Demand</div>
			<div class="col-span-2">
				{attrPIHeatingDemand.value.toLocaleString(undefined, { maximumFractionDigits: 0 })}%
			</div>
		{/if}
		<OthersContent others={attrOthers} {serviceAction}/>
	</div>
</ServiceRoot>
