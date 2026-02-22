<script lang="ts">
	import type {
		Attribute,
		BoolAttribute,
		FloatAttribute,
		IntAttribute,
		Service
	} from '$lib/api/v1/clients/client_service_pb';
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { UserRoundCheckIcon, UserRoundXIcon } from '@lucide/svelte';
	import { OthersContent } from '$lib/components/wh/attributes';

	let { deviceID, service, ...rest }: StandardProps = $props();

	let motion: boolean = $state(false);
	let presence: boolean = $state(false);
	let distance: number = $state(0.0);
	let attrOthers: Attribute[] = $state([]);

	$effect(() => {
		let others: Attribute[] = [];
		for (const attr of service.attrs) {
			if (attr.id === 'motion') {
				motion = attr.bool?.value!;
			} else if (attr.id === 'presence') {
				presence = attr.bool?.value!;
			} else if (attr.id === 'distance') {
				distance = attr.float?.value!;
			} else {
				others = [...others, attr];
			}
		}
		attrOthers = others;
	});

	let serviceAction = new ServiceAction(deviceID, service.id);
</script>

{#snippet icon()}
	{#if presence}
		<UserRoundCheckIcon />
	{:else}
		<UserRoundXIcon />
	{/if}
{/snippet}

{#snippet details()}
	<p>{presence ? 'Presence' : 'No Presence'}</p>
	<p class="text-muted-foreground">{motion ? 'Motion' : 'No Motion'}</p>
	<p class="text-muted-foreground">{distance.toLocaleString(undefined, { maximumFractionDigits: 1 })}m</p>
{/snippet}

<ServiceRoot {deviceID} {...rest} {service} {icon} {details}>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		<div>Presence</div>
		<div class="col-span-2">
			{#if presence}
				<p>Yes</p>
			{:else}
				<p>No</p>
			{/if}
		</div>
		<div>Motion</div>
		<div class="col-span-2">
			{#if motion}
				<p>Yes</p>
			{:else}
				<p>No</p>
			{/if}
		</div>
		<div>Distance</div>
		<div class="col-span-2">
			<p>{distance.toLocaleString(undefined, { maximumFractionDigits: 1 })}m</p>
		</div>
	</div>
	<OthersContent others={attrOthers} {serviceAction} />
</ServiceRoot>
