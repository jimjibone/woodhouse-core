<script lang="ts">
	import {
		BoolValueSchema,
		ValueSchema,
		type Attribute,
		type BoolAttribute,
		type DurationAttribute,
		type IntAttribute,
		type TextAttribute
	} from '$lib/api/v1/clients/client_service_pb';
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { PackagePlusIcon, PackageCheckIcon } from '@lucide/svelte';
	import { OthersContent } from '$lib/components/wh/attributes';
	import { toHumanDuration } from '$lib/tools/duration';
	import { create } from '@bufbuild/protobuf';
	import Button from '@/components/ui/button/button.svelte';

	let { deviceID, service, ...rest }: StandardProps = $props();

	let attrAvailable: BoolAttribute | undefined = $state(undefined);
	let attrUpdating: BoolAttribute | undefined = $state(undefined);
	let attrCurrentVersion: TextAttribute | undefined = $state(undefined);
	let attrUpdateVersion: TextAttribute | undefined = $state(undefined);
	let attrProgress: IntAttribute | undefined = $state(undefined);
	let attrRemaining: DurationAttribute | undefined = $state(undefined);
	let attrOthers: Attribute[] = $state([]);
	let available: boolean = $state(false);
	let updating: boolean = $state(false);

	$effect(() => {
		let others: Attribute[] = [];
		for (const attr of service.attrs) {
			if (attr.id === 'available') {
				attrAvailable = attr.bool;
				available = attr.bool?.value!;
			} else if (attr.id === 'updating') {
				attrUpdating = attr.bool;
				updating = attr.bool?.value!;
			} else if (attr.id === 'current_version') {
				attrCurrentVersion = attr.text;
			} else if (attr.id === 'update_version') {
				attrUpdateVersion = attr.text;
			} else if (attr.id === 'progress') {
				attrProgress = attr.int;
			} else if (attr.id === 'remaining') {
				attrRemaining = attr.duration;
			} else {
				others = [...others, attr];
			}
		}
		attrOthers = others;
	});

	let serviceAction = new ServiceAction(deviceID, service.id);

	const sendActionStartUpdate = async () => {
		serviceAction.send([
			create(ValueSchema, {
				id: 'start_update',
				bool: create(BoolValueSchema, { value: true })
			})
		]);
	};
</script>

{#snippet icon()}
	{#if available || updating}
		<PackagePlusIcon />
	{:else}
		<PackageCheckIcon />
	{/if}
{/snippet}

{#snippet details()}
	{#if available}
		<p>Update Available</p>
	{:else if updating}
		{#if attrProgress && attrRemaining}
			<p>Updating ({attrProgress.value}% {toHumanDuration(Number(attrRemaining.value))})</p>
		{:else}
			<p>Updating</p>
		{/if}
	{:else}
		<p>Up to Date</p>
	{/if}
{/snippet}

<ServiceRoot
	{deviceID}
	{...rest}
	{service}
	{icon}
	iconclass={available ? 'bg-green-400 text-black' : updating ? 'bg-amber-400 text-black' : false}
	{details}
>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if attrAvailable !== undefined}
			<div>Available</div>
			<div class="col-span-2">
				{#if attrAvailable.value}
					<Button class={'cursor-pointer'} onclick={() => sendActionStartUpdate()}>Start Update</Button>
				{:else if updating}
					<Button class={'cursor-pointer'} disabled>Updating...</Button>
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
		{#if attrProgress !== undefined}
			<div>Progress</div>
			<div class="col-span-2">
				{attrProgress.value}%
			</div>
		{/if}
		{#if attrRemaining !== undefined}
			<div>Remaining</div>
			<div class="col-span-2">
				{toHumanDuration(Number(attrRemaining.value))}
			</div>
		{/if}
		<OthersContent others={attrOthers} {serviceAction} />
	</div>
</ServiceRoot>
