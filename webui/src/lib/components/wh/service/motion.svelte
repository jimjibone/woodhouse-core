<script lang="ts">
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { RadarIcon } from '@lucide/svelte';
	import { OthersContent } from '$lib/components/wh/attributes';
	import { attrBool } from '$lib/device';
	import { Motion } from '$lib/schema/zz_services';

	let { deviceID, endpointID, service, ...rest }: StandardProps = $props();

	let motion = $derived(attrBool(service, Motion.attributes.motion));

	let serviceAction = new ServiceAction(deviceID, endpointID, service.id);
</script>

{#snippet icon()}
	<RadarIcon />
{/snippet}

{#snippet details()}
	{#if motion !== undefined}
		<p>{motion ? 'Motion' : 'No Motion'}</p>
	{/if}
{/snippet}

<ServiceRoot
	{deviceID}
	{endpointID}
	{...rest}
	{service}
	{icon}
	iconclass={motion ? 'bg-amber-400 text-black' : false}
	{details}
>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if motion !== undefined}
			<div>Motion</div>
			<div class="col-span-2">
				<p>{motion ? 'Yes' : 'No'}</p>
			</div>
		{/if}
	</div>
	<OthersContent
		{deviceID}
		{endpointID}
		{service}
		action={serviceAction}
		exclude={[Motion.attributes.motion]}
	/>
</ServiceRoot>
