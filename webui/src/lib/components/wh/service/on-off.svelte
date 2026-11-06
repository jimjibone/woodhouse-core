<script lang="ts">
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { PowerIcon } from '@lucide/svelte';
	import { BoolContent, OthersContent } from '$lib/components/wh/attributes';
	import { attrBool, boolValue } from '$lib/device';
	import { OnOff } from '$lib/schema/zz_services';

	let { deviceID, endpointID, service, ...rest }: StandardProps = $props();

	const on = $derived(attrBool(service, OnOff.attributes.on));

	// svelte-ignore state_referenced_locally
	const serviceAction = new ServiceAction(deviceID, endpointID, service.id);

	const sendOn = (val: boolean) => serviceAction.writeValue(OnOff.attributes.on, boolValue(val));

	const oniconclick = () => {
		if (on !== undefined) {
			sendOn(!on);
		}
	};
</script>

{#snippet icon()}
	<PowerIcon />
{/snippet}

{#snippet details()}
	{#if on !== undefined}
		<p>{on ? 'On' : 'Off'}</p>
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
	{oniconclick}
	iconclass={on ? 'bg-green-400 text-black' : false}
	{details}
>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if on !== undefined}
			<BoolContent name="On" value={on} onaction={sendOn} />
		{/if}
	</div>
	<!-- The toggle command, if the service has one, is rendered as a button. -->
	<OthersContent {deviceID} {endpointID} {service} action={serviceAction} exclude={[OnOff.attributes.on]} />
</ServiceRoot>
