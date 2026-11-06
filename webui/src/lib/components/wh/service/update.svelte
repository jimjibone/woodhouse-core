<script lang="ts">
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { PackagePlusIcon, PackageCheckIcon } from '@lucide/svelte';
	import { OthersContent } from '$lib/components/wh/attributes';
	import { toHumanDuration } from '$lib/tools/duration';
	import { attrBool, attrDuration, attrInt, attrText, findCommand } from '$lib/device';
	import { Update } from '$lib/schema/zz_services';
	import Button from '@/components/ui/button/button.svelte';

	let { deviceID, endpointID, service, ...rest }: StandardProps = $props();

	let available = $derived(attrBool(service, Update.attributes.available));
	let updating = $derived(attrBool(service, Update.attributes.updating));
	let currentVersion = $derived(attrText(service, Update.attributes.currentVersion));
	let updateVersion = $derived(attrText(service, Update.attributes.updateVersion));
	let progress = $derived(attrInt(service, Update.attributes.progress));
	let remaining = $derived(attrDuration(service, Update.attributes.remaining));
	let canStart = $derived(findCommand(service, Update.commands.start) !== undefined);

	let serviceAction = new ServiceAction(deviceID, endpointID, service.id);

	const startUpdate = () => serviceAction.command(Update.commands.start);
</script>

{#snippet icon()}
	{#if available || updating}
		<PackagePlusIcon />
	{:else}
		<PackageCheckIcon />
	{/if}
{/snippet}

{#snippet details()}
	{#if updating}
		{#if progress !== undefined && remaining !== undefined}
			<p>Updating ({progress}% {toHumanDuration(Number(remaining))})</p>
		{:else}
			<p>Updating</p>
		{/if}
	{:else if available}
		<p>Update Available</p>
	{:else}
		<p>Up to Date</p>
	{/if}
{/snippet}

<ServiceRoot
	{deviceID}
	{endpointID}
	{...rest}
	{service}
	actionPending={serviceAction.pending}
	errorSignal={serviceAction.error}
	{icon}
	iconclass={available ? 'bg-green-400 text-black' : updating ? 'bg-amber-400 text-black' : false}
	{details}
>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if available !== undefined}
			<div>Available</div>
			<div class="col-span-2">
				{#if updating}
					<Button class={'cursor-pointer'} disabled>Updating...</Button>
				{:else if available && canStart}
					<Button class={'cursor-pointer'} onclick={startUpdate}>Start Update</Button>
				{:else}
					<p>{available ? 'Yes' : 'No'}</p>
				{/if}
			</div>
		{/if}
		{#if updateVersion !== undefined}
			<div>Update Version</div>
			<div class="col-span-2">
				{updateVersion}
			</div>
		{/if}
		{#if currentVersion !== undefined}
			<div>Current Version</div>
			<div class="col-span-2">
				{currentVersion}
			</div>
		{/if}
		{#if progress !== undefined}
			<div>Progress</div>
			<div class="col-span-2">
				{progress}%
			</div>
		{/if}
		{#if remaining !== undefined}
			<div>Remaining</div>
			<div class="col-span-2">
				{toHumanDuration(Number(remaining))}
			</div>
		{/if}
	</div>
	<OthersContent
		{deviceID}
		{endpointID}
		{service}
		action={serviceAction}
		exclude={[
			Update.attributes.available,
			Update.attributes.updating,
			Update.attributes.currentVersion,
			Update.attributes.updateVersion,
			Update.attributes.progress,
			Update.attributes.remaining
		]}
		excludeCommands={[Update.commands.start]}
	/>
</ServiceRoot>
