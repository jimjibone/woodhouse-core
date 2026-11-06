<script lang="ts">
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { SearchIcon } from '@lucide/svelte';
	import { Button } from '$lib/components/ui/button';
	import { OthersContent } from '$lib/components/wh/attributes';
	import { attrBool, findCommand } from '$lib/device';
	import { Identify } from '$lib/schema/zz_services';

	let { deviceID, endpointID, service, ...rest }: StandardProps = $props();

	const identifying = $derived(attrBool(service, Identify.attributes.identifying));
	const canIdentify = $derived(findCommand(service, Identify.commands.identify) !== undefined);

	// svelte-ignore state_referenced_locally
	const serviceAction = new ServiceAction(deviceID, endpointID, service.id);

	// No duration is sent, so the device uses its default.
	const identify = () => serviceAction.command(Identify.commands.identify);
</script>

{#snippet icon()}
	<SearchIcon />
{/snippet}

{#snippet details()}
	<p>{identifying ? 'Identifying' : 'Idle'}</p>
{/snippet}

<ServiceRoot
	{deviceID}
	{endpointID}
	{...rest}
	{service}
	actionPending={serviceAction.pending}
	errorSignal={serviceAction.error}
	{icon}
	oniconclick={canIdentify ? identify : undefined}
	iconclass={identifying ? 'bg-amber-400 text-black' : false}
	{details}
>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if identifying !== undefined}
			<div>Identifying</div>
			<div class="col-span-2">{identifying ? 'Yes' : 'No'}</div>
		{/if}
	</div>
	{#if canIdentify}
		<div class="pt-4">
			<Button class="cursor-pointer" onclick={identify}>Identify</Button>
		</div>
	{/if}
	<OthersContent
		{deviceID}
		{endpointID}
		{service}
		action={serviceAction}
		exclude={[Identify.attributes.identifying]}
		excludeCommands={[Identify.commands.identify]}
	/>
</ServiceRoot>
