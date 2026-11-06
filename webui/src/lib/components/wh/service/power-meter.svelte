<script lang="ts">
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { ZapIcon } from '@lucide/svelte';
	import { OthersContent } from '$lib/components/wh/attributes';
	import { findAttr, formatValue } from '$lib/device';
	import { PowerMeter } from '$lib/schema/zz_services';

	let { deviceID, endpointID, service, ...rest }: StandardProps = $props();

	// Read-only measurements in display order; units come from the specs.
	const readings = [
		{ id: PowerMeter.attributes.power, label: 'Power' },
		{ id: PowerMeter.attributes.current, label: 'Current' },
		{ id: PowerMeter.attributes.voltage, label: 'Voltage' },
		{ id: PowerMeter.attributes.energy, label: 'Energy' }
	];

	const present = $derived(
		readings.flatMap((r) => {
			const attr = findAttr(service, r.id);
			return attr === undefined ? [] : [{ ...r, text: formatValue(attr.value, attr.spec) }];
		})
	);

	// svelte-ignore state_referenced_locally
	const serviceAction = new ServiceAction(deviceID, endpointID, service.id);
</script>

{#snippet icon()}
	<ZapIcon />
{/snippet}

{#snippet details()}
	{#each present.slice(0, 2) as reading, i (reading.id)}
		<p class={i > 0 ? 'text-muted-foreground' : undefined}>{reading.text === '' ? '-' : reading.text}</p>
	{/each}
{/snippet}

<ServiceRoot {deviceID} {endpointID} {...rest} {service} {icon} {details}>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#each present as reading (reading.id)}
			<div>{reading.label}</div>
			<div class="col-span-2">{reading.text === '' ? '-' : reading.text}</div>
		{/each}
	</div>
	<OthersContent
		{deviceID}
		{endpointID}
		{service}
		action={serviceAction}
		exclude={readings.map((r) => r.id)}
	/>
</ServiceRoot>
