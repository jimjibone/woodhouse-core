<script lang="ts">
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { LogInIcon } from '@lucide/svelte';
	import { OthersContent } from '$lib/components/wh/attributes';
	import { attrBool } from '$lib/device';
	import { BinaryInput } from '$lib/schema/zz_services';

	let { deviceID, endpointID, service, ...rest }: StandardProps = $props();

	const on = $derived(attrBool(service, BinaryInput.attributes.on));

	// svelte-ignore state_referenced_locally
	const serviceAction = new ServiceAction(deviceID, endpointID, service.id);
</script>

{#snippet icon()}
	<LogInIcon />
{/snippet}

{#snippet details()}
	{#if on !== undefined}
		<p>{on ? 'On' : 'Off'}</p>
	{/if}
{/snippet}

<ServiceRoot
	{deviceID}
	{endpointID}
	{...rest}
	{service}
	{icon}
	iconclass={on ? 'bg-green-400 text-black' : false}
	{details}
>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if on !== undefined}
			<div>On</div>
			<div class="col-span-2">
				<p>{on ? 'Yes' : 'No'}</p>
			</div>
		{/if}
	</div>
	<OthersContent
		{deviceID}
		{endpointID}
		{service}
		action={serviceAction}
		exclude={[BinaryInput.attributes.on]}
	/>
</ServiceRoot>
