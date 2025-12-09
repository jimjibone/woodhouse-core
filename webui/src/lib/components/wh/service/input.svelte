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
	import { LogInIcon } from '@lucide/svelte';
	import { OthersContent } from '$lib/components/wh/attributes';

	let { deviceID, service, ...rest }: StandardProps = $props();

	let attrOn: BoolAttribute | undefined = $state(undefined);
	let attrOthers: Attribute[] = $state([]);
	let on: boolean = $state(false);

	$effect(() => {
		let others: Attribute[] = [];
		for (const attr of service.attrs) {
			if (attr.id === 'on') {
				attrOn = attr.bool;
				on = attr.bool?.value!;
			} else {
				others = [...others, attr];
			}
		}
		attrOthers = others;
	});

	let serviceAction = new ServiceAction(deviceID, service.id);
</script>

{#snippet icon()}
	<LogInIcon />
{/snippet}

{#snippet details()}
	{#if attrOn !== undefined}
		{#if attrOn.value}
			<p>On</p>
		{:else}
			<p>Off</p>
		{/if}
	{/if}
{/snippet}

<ServiceRoot {deviceID} {...rest} {service} {icon} iconclass={on ? 'bg-green-400 text-black' : false} {details}>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if attrOn !== undefined}
			<div>On</div>
			<div class="col-span-2">
				{#if attrOn.value}
					<p>Yes</p>
				{:else}
					<p>No</p>
				{/if}
			</div>
		{/if}
	</div>
	<OthersContent others={attrOthers} {serviceAction} />
</ServiceRoot>
