<script lang="ts">
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { TvIcon } from '@lucide/svelte';
	import { Button } from '$lib/components/ui/button';
	import { OthersContent } from '$lib/components/wh/attributes';
	import { enumValue, field, findCommand, specEnumOptions } from '$lib/device';
	import { toHeadlineCase } from '$lib/tools/headline-case';
	import { Remote } from '$lib/schema/zz_services';

	let { deviceID, endpointID, service, ...rest }: StandardProps = $props();

	// The keys the device accepts are set by the bridge, so they come from the
	// spec of the key parameter.
	const keys = $derived(
		specEnumOptions(findCommand(service, Remote.commands.sendKey)?.params.find((p) => p.id === 'key')?.spec)
	);

	// svelte-ignore state_referenced_locally
	const serviceAction = new ServiceAction(deviceID, endpointID, service.id);

	const sendKey = (key: string) => serviceAction.command(Remote.commands.sendKey, [field('key', enumValue(key))]);
</script>

{#snippet icon()}
	<TvIcon />
{/snippet}

{#snippet details()}
	<p>{keys.length} {keys.length === 1 ? 'key' : 'keys'}</p>
{/snippet}

<ServiceRoot
	{deviceID}
	{endpointID}
	{...rest}
	{service}
	actionPending={serviceAction.pending}
	errorSignal={serviceAction.error}
	{icon}
	{details}
>
	{#if keys.length === 0}
		<p class="text-muted-foreground">No keys</p>
	{:else}
		<div class="flex flex-row flex-wrap gap-2">
			{#each keys as key (key)}
				<Button variant="secondary" class="cursor-pointer" onclick={() => sendKey(key)}>
					{toHeadlineCase(key)}
				</Button>
			{/each}
		</div>
	{/if}
	<OthersContent
		{deviceID}
		{endpointID}
		{service}
		action={serviceAction}
		excludeCommands={[Remote.commands.sendKey]}
	/>
</ServiceRoot>
