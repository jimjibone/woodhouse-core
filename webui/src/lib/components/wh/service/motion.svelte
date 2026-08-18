<script lang="ts">
	import type { Attribute, BoolAttribute } from '$lib/api/v1/clients/client_service_pb';
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { RadarIcon } from '@lucide/svelte';
	import { OthersContent } from '$lib/components/wh/attributes';

	let { deviceID, service, ...rest }: StandardProps = $props();

	let attrMotion: BoolAttribute | undefined = $state(undefined);
	let attrOthers: Attribute[] = $state([]);
	let motion: boolean = $state(false);

	$effect(() => {
		let others: Attribute[] = [];
		for (const attr of service.attrs) {
			if (attr.id === 'motion') {
				attrMotion = attr.bool;
				motion = attr.bool?.value!;
			} else {
				others = [...others, attr];
			}
		}
		attrOthers = others;
	});

	let serviceAction = new ServiceAction(deviceID, service.id);
</script>

{#snippet icon()}
	<RadarIcon />
{/snippet}

{#snippet details()}
	{#if attrMotion !== undefined}
		{#if attrMotion.value}
			<p>Motion</p>
		{:else}
			<p>No Motion</p>
		{/if}
	{/if}
{/snippet}

<ServiceRoot {deviceID} {...rest} {service} {icon} iconclass={motion ? 'bg-amber-400 text-black' : false} {details}>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if attrMotion !== undefined}
			<div>Motion</div>
			<div class="col-span-2">
				{#if attrMotion.value}
					<p>Yes</p>
				{:else}
					<p>No</p>
				{/if}
			</div>
		{/if}
	</div>
	<OthersContent others={attrOthers} {serviceAction} />
</ServiceRoot>
