<script lang="ts">
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { Rows3Icon } from '@lucide/svelte';
	import { EnumContent, OthersContent } from '$lib/components/wh/attributes';
	import { attrEnum, enumValue, findAttr, isWritable, specEnumOptions } from '$lib/device';
	import { Mode } from '$lib/schema/zz_services';

	let { deviceID, endpointID, service, ...rest }: StandardProps = $props();

	const valueAttr = $derived(findAttr(service, Mode.attributes.value));
	const value = $derived(attrEnum(service, Mode.attributes.value));
	// The options are set by the provider, so they come from the spec.
	const options = $derived(specEnumOptions(valueAttr?.spec));
	const writable = $derived(valueAttr !== undefined && isWritable(valueAttr));

	// svelte-ignore state_referenced_locally
	const serviceAction = new ServiceAction(deviceID, endpointID, service.id);

	const sendValue = (val: string) => serviceAction.writeValue(Mode.attributes.value, enumValue(val));
</script>

{#snippet icon()}
	<Rows3Icon />
{/snippet}

{#snippet details()}
	{#if valueAttr !== undefined}
		<p class="whitespace-pre">{value ? value : 'No state'}</p>
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
	{details}
>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if valueAttr !== undefined}
			<EnumContent name="Value" {value} {options} onaction={writable ? sendValue : undefined} />
		{/if}
	</div>
	<OthersContent
		{deviceID}
		{endpointID}
		{service}
		action={serviceAction}
		exclude={[Mode.attributes.value]}
	/>
</ServiceRoot>
