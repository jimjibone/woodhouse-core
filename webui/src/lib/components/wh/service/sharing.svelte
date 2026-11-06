<script lang="ts">
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { Share2Icon } from '@lucide/svelte';
	import { Button } from '$lib/components/ui/button';
	import { OthersContent } from '$lib/components/wh/attributes';
	import { attrBool, attrText, attrTime, findCommand } from '$lib/device';
	import { Sharing } from '$lib/schema/zz_services';

	let { deviceID, endpointID, service, ...rest }: StandardProps = $props();

	const open = $derived(attrBool(service, Sharing.attributes.open));
	const code = $derived(attrText(service, Sharing.attributes.code));
	const qr = $derived(attrText(service, Sharing.attributes.qr));
	const ends = $derived(attrTime(service, Sharing.attributes.ends));
	const canOpen = $derived(findCommand(service, Sharing.commands.open) !== undefined);
	const canClose = $derived(findCommand(service, Sharing.commands.close) !== undefined);

	// svelte-ignore state_referenced_locally
	const serviceAction = new ServiceAction(deviceID, endpointID, service.id);

	// No timeout is sent, so the device uses its default.
	const openSharing = () => serviceAction.command(Sharing.commands.open);
	const closeSharing = () => serviceAction.command(Sharing.commands.close);
</script>

{#snippet icon()}
	<Share2Icon />
{/snippet}

{#snippet details()}
	{#if open !== undefined}
		<p>{open ? 'Open' : 'Closed'}</p>
	{/if}
	{#if open && code}
		<p class="text-muted-foreground font-mono">{code}</p>
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
	iconclass={open ? 'bg-green-400 text-black' : false}
	{details}
>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if open !== undefined}
			<div>Open</div>
			<div class="col-span-2">{open ? 'Yes' : 'No'}</div>
		{/if}
		{#if code}
			<div>Code</div>
			<div class="col-span-2 font-mono break-all">{code}</div>
		{/if}
		{#if qr}
			<div>QR</div>
			<div class="col-span-2 font-mono break-all">{qr}</div>
		{/if}
		{#if ends !== undefined}
			<div>Ends</div>
			<div class="col-span-2">{ends.toLocaleString()}</div>
		{/if}
	</div>
	{#if canOpen || canClose}
		<div class="flex flex-row gap-2 pt-4">
			{#if canOpen}
				<Button class="cursor-pointer" onclick={openSharing}>{open ? 'Reopen' : 'Open'}</Button>
			{/if}
			{#if canClose}
				<Button variant="secondary" class="cursor-pointer" disabled={open === false} onclick={closeSharing}>
					Close
				</Button>
			{/if}
		</div>
	{/if}
	<OthersContent
		{deviceID}
		{endpointID}
		{service}
		action={serviceAction}
		exclude={[
			Sharing.attributes.open,
			Sharing.attributes.code,
			Sharing.attributes.qr,
			Sharing.attributes.ends
		]}
		excludeCommands={[Sharing.commands.open, Sharing.commands.close]}
	/>
</ServiceRoot>
