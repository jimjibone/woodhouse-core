<script lang="ts">
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { DoorOpenIcon, DoorClosedIcon } from '@lucide/svelte';
	import { OthersContent } from '$lib/components/wh/attributes';
	import { attrBool } from '$lib/device';
	import { Contact } from '$lib/schema/zz_services';

	let { deviceID, endpointID, service, ...rest }: StandardProps = $props();

	let closed = $derived(attrBool(service, Contact.attributes.closed));

	let serviceAction = new ServiceAction(deviceID, endpointID, service.id);
</script>

{#snippet icon()}
	{#if closed === false}
		<DoorOpenIcon />
	{:else}
		<DoorClosedIcon />
	{/if}
{/snippet}

{#snippet details()}
	{#if closed !== undefined}
		<p>{closed ? 'Closed' : 'Open'}</p>
	{/if}
{/snippet}

<ServiceRoot {deviceID} {endpointID} {...rest} {service} {icon} {details}>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if closed !== undefined}
			<div>Closed</div>
			<div class="col-span-2">
				<p>{closed ? 'Yes' : 'No'}</p>
			</div>
		{/if}
	</div>
	<OthersContent
		{deviceID}
		{endpointID}
		{service}
		action={serviceAction}
		exclude={[Contact.attributes.closed]}
	/>
</ServiceRoot>
