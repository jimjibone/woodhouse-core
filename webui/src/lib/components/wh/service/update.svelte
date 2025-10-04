<script lang="ts">
	import type { Attribute, BoolAttribute, FloatAttribute, IntAttribute, Service, TextAttribute } from '$lib/api/v1/clients/client_service_pb';
	import ServiceRoot, { type StandardProps } from "./service-root.svelte";
	import ServiceAction from './service-action.svelte';
	import { PackagePlusIcon, PackageCheckIcon } from '@lucide/svelte';
	import { OthersContent } from '$lib/components/wh/attributes';

	let {
		deviceID,
		service,
		...rest
	}: StandardProps = $props();

	let attrAvailable: BoolAttribute | undefined = $state(undefined);
	let attrCurrentVersion: TextAttribute | undefined = $state(undefined);
	let attrUpdateVersion: TextAttribute | undefined = $state(undefined);
	let attrOthers: Attribute[] = $state([]);
	let available: boolean = $state(false);

	$effect(() => {
		let others: Attribute[] = [];
		for (const attr of service.attrs) {
			if (attr.id === 'available') {
				attrAvailable = attr.bool;
				available = attr.bool?.value!;
			} else if (attr.id === 'current_version') {
				attrCurrentVersion = attr.text;
			} else  if (attr.id === 'update_version') {
				attrUpdateVersion = attr.text;
			} else {
				others = [...others, attr];
			}
		}
		attrOthers = others;
	});

	let serviceAction = new ServiceAction(deviceID, service.id);
</script>

{#snippet icon()}
	{#if available}
		<PackagePlusIcon/>
	{:else}
		<PackageCheckIcon/>
	{/if}
{/snippet}

{#snippet details()}
	{#if attrAvailable !== undefined}
		{#if attrAvailable.value}
			<p>Update Available</p>
		{:else}
			<p>Up to Date</p>
		{/if}
	{/if}
{/snippet}

<ServiceRoot
	{deviceID}
	{...rest}
	service={service}
	icon={icon}
	iconclass={available ? "bg-green-400 text-black" : false}
	details={details}
>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if attrAvailable !== undefined}
			<div>Available</div>
			<div class="col-span-2">
				{#if attrAvailable.value}
					<p>Yes</p>
				{:else}
					<p>No</p>
				{/if}
			</div>
		{/if}
		{#if attrUpdateVersion !== undefined}
			<div>Update Version</div>
			<div class="col-span-2">
				{attrUpdateVersion.value}
			</div>
		{/if}
		{#if attrCurrentVersion !== undefined}
			<div>Current Version</div>
			<div class="col-span-2">
				{attrCurrentVersion.value}
			</div>
		{/if}
		<OthersContent others={attrOthers} {serviceAction}/>
	</div>
</ServiceRoot>
