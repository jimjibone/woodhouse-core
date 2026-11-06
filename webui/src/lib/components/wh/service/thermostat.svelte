<script lang="ts">
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { FlameIcon } from '@lucide/svelte';
	import { FloatContent, IntContent, OthersContent } from '$lib/components/wh/attributes';
	import {
		attrBool,
		attrFloat,
		attrInt,
		findAttr,
		floatValue,
		formatValue,
		intValue,
		isWritable,
		specRange,
		specUnit
	} from '$lib/device';
	import { unitLabel } from '$lib/tools/units';
	import { Thermostat } from '$lib/schema/zz_services';

	let { deviceID, endpointID, service, ...rest }: StandardProps = $props();

	const setpointAttr = $derived(findAttr(service, Thermostat.attributes.heatingSetpoint));
	const localTempAttr = $derived(findAttr(service, Thermostat.attributes.localTemperature));
	const demandAttr = $derived(findAttr(service, Thermostat.attributes.heatingDemand));
	const piDemandAttr = $derived(findAttr(service, Thermostat.attributes.piHeatingDemand));
	const valveAttr = $derived(findAttr(service, Thermostat.attributes.valvePosition));

	const setpoint = $derived(attrFloat(service, Thermostat.attributes.heatingSetpoint));
	const heating = $derived(attrBool(service, Thermostat.attributes.heatingDemand));
	const piDemand = $derived(attrInt(service, Thermostat.attributes.piHeatingDemand));
	const valve = $derived(attrInt(service, Thermostat.attributes.valvePosition));

	const setpointRange = $derived(specRange(setpointAttr?.spec));
	const setpointStep = $derived(setpointRange.step ?? 0.5);
	const setpointDigits = $derived(setpointStep < 1 ? 1 : 0);
	const setpointUnits = $derived(unitLabel(specUnit(setpointAttr?.spec)));
	const valveRange = $derived(specRange(valveAttr?.spec));

	const text = (attr: typeof setpointAttr) => (attr === undefined ? '' : formatValue(attr.value, attr.spec));

	// The schedule list and the set_schedule command stay with the generic
	// rendering.
	// svelte-ignore state_referenced_locally
	const serviceAction = new ServiceAction(deviceID, endpointID, service.id);

	const sendSetpoint = (val: number) => serviceAction.writeValue(Thermostat.attributes.heatingSetpoint, floatValue(val));
	const sendValve = (val: bigint) => serviceAction.writeValue(Thermostat.attributes.valvePosition, intValue(val));
</script>

{#snippet icon()}
	<FlameIcon />
{/snippet}

{#snippet details()}
	{#if setpointAttr !== undefined}
		<p>{text(setpointAttr)}</p>
	{/if}
	{#if localTempAttr !== undefined}
		<p class="text-muted-foreground">{text(localTempAttr)}</p>
	{/if}
	{#if piDemandAttr !== undefined}
		<p class="text-muted-foreground">{text(piDemandAttr)}</p>
	{/if}
{/snippet}

<ServiceRoot
	{deviceID}
	{endpointID}
	{...rest}
	{service}
	actionPending={serviceAction.pending}
	errorSignal={serviceAction.error}
	{icon}
	iconclass={heating ? 'bg-amber-400 text-black' : false}
	{details}
>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if setpointAttr !== undefined}
			{#if isWritable(setpointAttr)}
				<FloatContent
					name="Target"
					value={setpoint}
					min={setpointRange.min ?? 5}
					max={setpointRange.max ?? 30}
					step={setpointStep}
					onaction={sendSetpoint}
					maximumFractionDigits={setpointDigits}
					units={setpointUnits}
				/>
			{:else}
				<div>Target</div>
				<div class="col-span-2">{text(setpointAttr)}</div>
			{/if}
		{/if}
		{#if localTempAttr !== undefined}
			<div>Current</div>
			<div class="col-span-2">{text(localTempAttr)}</div>
		{/if}
		{#if demandAttr !== undefined}
			<div>Heating</div>
			<div class="col-span-2">{heating === undefined ? '-' : heating ? 'Yes' : 'No'}</div>
		{/if}
		{#if piDemandAttr !== undefined}
			<div>Demand</div>
			<div class="col-span-2">{piDemand === undefined ? '-' : text(piDemandAttr)}</div>
		{/if}
		{#if valveAttr !== undefined}
			{#if isWritable(valveAttr)}
				<IntContent
					name="Valve"
					value={valve}
					min={BigInt(valveRange.min ?? 0)}
					max={BigInt(valveRange.max ?? 100)}
					step={BigInt(Math.max(1, valveRange.step ?? 1))}
					onaction={sendValve}
					units="%"
				/>
			{:else}
				<div>Valve</div>
				<div class="col-span-2">{valve === undefined ? '-' : text(valveAttr)}</div>
			{/if}
		{/if}
	</div>
	<OthersContent
		{deviceID}
		{endpointID}
		{service}
		action={serviceAction}
		exclude={[
			Thermostat.attributes.heatingSetpoint,
			Thermostat.attributes.localTemperature,
			Thermostat.attributes.heatingDemand,
			Thermostat.attributes.piHeatingDemand,
			Thermostat.attributes.valvePosition
		]}
	/>
</ServiceRoot>
