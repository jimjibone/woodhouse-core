<script lang="ts">
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { UserRoundCheckIcon, UserRoundXIcon } from '@lucide/svelte';
	import { OthersContent } from '$lib/components/wh/attributes';
	import { attrBool, attrFloat } from '$lib/device';
	import { Presence } from '$lib/schema/zz_services';

	let { deviceID, endpointID, service, ...rest }: StandardProps = $props();

	let presence = $derived(attrBool(service, Presence.attributes.presence));
	// Distance is only there when the device reports it (feature "distance").
	let distance = $derived(attrFloat(service, Presence.attributes.distance));

	let serviceAction = new ServiceAction(deviceID, endpointID, service.id);
</script>

{#snippet icon()}
	{#if presence}
		<UserRoundCheckIcon />
	{:else}
		<UserRoundXIcon />
	{/if}
{/snippet}

{#snippet details()}
	{#if presence !== undefined}
		<p>{presence ? 'Presence' : 'No Presence'}</p>
	{/if}
	{#if distance !== undefined}
		<p class="text-muted-foreground">{distance.toLocaleString(undefined, { maximumFractionDigits: 1 })}m</p>
	{/if}
{/snippet}

<ServiceRoot {deviceID} {endpointID} {...rest} {service} {icon} {details}>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if presence !== undefined}
			<div>Presence</div>
			<div class="col-span-2">
				<p>{presence ? 'Yes' : 'No'}</p>
			</div>
		{/if}
		{#if distance !== undefined}
			<div>Distance</div>
			<div class="col-span-2">
				<p>{distance.toLocaleString(undefined, { maximumFractionDigits: 1 })}m</p>
			</div>
		{/if}
	</div>
	<OthersContent
		{deviceID}
		{endpointID}
		{service}
		action={serviceAction}
		exclude={[Presence.attributes.presence, Presence.attributes.distance]}
	/>
</ServiceRoot>
