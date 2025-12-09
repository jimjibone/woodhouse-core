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
	import { DoorOpenIcon, DoorClosedIcon } from '@lucide/svelte';
	import { OthersContent } from '$lib/components/wh/attributes';

	let { deviceID, service, ...rest }: StandardProps = $props();

	let attrClosed: BoolAttribute | undefined = $state(undefined);
	let attrOthers: Attribute[] = $state([]);
	let closed: boolean = $state(false);

	$effect(() => {
		let others: Attribute[] = [];
		for (const attr of service.attrs) {
			if (attr.id === 'closed') {
				attrClosed = attr.bool;
				closed = attr.bool?.value!;
			} else {
				others = [...others, attr];
			}
		}
		attrOthers = others;
	});

	let serviceAction = new ServiceAction(deviceID, service.id);
</script>

{#snippet icon()}
	{#if closed}
		<DoorClosedIcon />
	{:else}
		<DoorOpenIcon />
	{/if}
{/snippet}

{#snippet details()}
	{#if attrClosed !== undefined}
		{#if attrClosed.value}
			<p>Closed</p>
		{:else}
			<p>Open</p>
		{/if}
	{/if}
{/snippet}

<ServiceRoot {deviceID} {...rest} {service} {icon} {details}>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if attrClosed !== undefined}
			<div>Closed</div>
			<div class="col-span-2">
				{#if attrClosed.value}
					<p>Yes</p>
				{:else}
					<p>No</p>
				{/if}
			</div>
		{/if}
	</div>
	<OthersContent others={attrOthers} {serviceAction} />
</ServiceRoot>
