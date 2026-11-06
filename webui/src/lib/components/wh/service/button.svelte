<script lang="ts">
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { PointerIcon } from '@lucide/svelte';
	import { EnumContent, OthersContent } from '$lib/components/wh/attributes';
	import { attrDuration, attrEnum, findAttr, specEnumOptions } from '$lib/device';
	import { toHeadlineCase } from '$lib/tools/headline-case';
	import { Button } from '$lib/schema/zz_services';

	let { deviceID, endpointID, service, ...rest }: StandardProps = $props();

	let lastEvent = $derived(attrEnum(service, Button.attributes.lastEvent));
	let lastDuration = $derived(attrDuration(service, Button.attributes.lastDuration));
	let eventOptions = $derived(specEnumOptions(findAttr(service, Button.attributes.lastEvent)?.spec));

	let serviceAction = new ServiceAction(deviceID, endpointID, service.id);
</script>

{#snippet icon()}
	<PointerIcon />
{/snippet}

{#snippet details()}
	{#if findAttr(service, Button.attributes.lastEvent) !== undefined}
		{#if !lastEvent}
			<p>No events</p>
		{:else}
			<p>{toHeadlineCase(lastEvent)}</p>
		{/if}
	{/if}
	{#if lastDuration !== undefined && lastDuration > 0n}
		<p class="text-muted-foreground">
			{lastDuration.toLocaleString(undefined, { maximumFractionDigits: 0 })}ms
		</p>
	{/if}
{/snippet}

<ServiceRoot {deviceID} {endpointID} {...rest} {service} {icon} {details}>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if findAttr(service, Button.attributes.lastEvent) !== undefined}
			<EnumContent name="Last Event" value={lastEvent} options={eventOptions} />
		{/if}
		{#if lastDuration !== undefined && lastDuration > 0n}
			<div>Last Duration</div>
			<div class="col-span-2">
				{lastDuration.toLocaleString(undefined, { maximumFractionDigits: 0 })}ms
			</div>
		{/if}
	</div>
	<OthersContent
		{deviceID}
		{endpointID}
		{service}
		action={serviceAction}
		exclude={[Button.attributes.lastEvent, Button.attributes.lastDuration]}
	/>
</ServiceRoot>
