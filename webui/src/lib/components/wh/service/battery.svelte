<script lang="ts">
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import {
		BatteryChargingIcon,
		BatteryFullIcon,
		BatteryLowIcon,
		BatteryMediumIcon,
		BatteryWarningIcon
	} from '@lucide/svelte';
	import { OthersContent } from '$lib/components/wh/attributes';
	import { attrBool, attrFloat, attrInt } from '$lib/device';
	import { Battery } from '$lib/schema/zz_services';

	let { deviceID, endpointID, service, ...rest }: StandardProps = $props();

	let attrLevel = $derived(attrInt(service, Battery.attributes.level));
	let attrVoltage = $derived(attrFloat(service, Battery.attributes.voltage));
	let charging = $derived(attrBool(service, Battery.attributes.charging));
	let level = $derived(attrLevel === undefined ? 100 : Number(attrLevel));

	let serviceAction = new ServiceAction(deviceID, endpointID, service.id);
</script>

{#snippet icon()}
	{#if charging}
		<BatteryChargingIcon />
	{:else if level < 20}
		<BatteryWarningIcon />
	{:else if level < 33}
		<BatteryLowIcon />
	{:else if level < 66}
		<BatteryMediumIcon />
	{:else}
		<BatteryFullIcon />
	{/if}
{/snippet}

{#snippet details()}
	{#if attrLevel !== undefined}
		<p>
			{attrLevel.toLocaleString(undefined, { maximumFractionDigits: 0 })}%
		</p>
	{/if}
	{#if attrVoltage !== undefined}
		<p class="text-muted-foreground">
			{attrVoltage.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })}V
		</p>
	{/if}
	{#if charging}
		<p class="text-muted-foreground">Charging</p>
	{/if}
{/snippet}

<ServiceRoot
	{deviceID}
	{endpointID}
	{...rest}
	{service}
	{icon}
	iconclass={level < 20 ? 'bg-red-400 text-black' : level < 33 ? 'bg-yellow-400 text-black' : false}
	{details}
>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if attrLevel !== undefined}
			<div>Level</div>
			<div class="col-span-2">
				{attrLevel.toLocaleString(undefined, { maximumFractionDigits: 0 })}%
			</div>
		{/if}
		{#if attrVoltage !== undefined}
			<div>Voltage</div>
			<div class="col-span-2">
				{attrVoltage.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })}V
			</div>
		{/if}
		{#if charging !== undefined}
			<div>Charging</div>
			<div class="col-span-2">{charging ? 'Yes' : 'No'}</div>
		{/if}
	</div>
	<OthersContent
		{deviceID}
		{endpointID}
		{service}
		action={serviceAction}
		exclude={[Battery.attributes.level, Battery.attributes.voltage, Battery.attributes.charging]}
	/>
</ServiceRoot>
