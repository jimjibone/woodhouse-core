<script lang="ts">
	// A read-only service whose main content is one attribute value with its
	// unit (temperature, humidity, pressure, ...). Other attributes fall
	// through to the generic rendering. The typed wrappers pick the icon and
	// the attribute.
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { OthersContent } from '$lib/components/wh/attributes';
	import { findAttr, formatValue } from '$lib/device';
	import { toHeadlineCase } from '$lib/tools/headline-case';
	import type { Component } from 'svelte';

	let {
		deviceID,
		endpointID,
		service,
		attrID,
		label = undefined,
		icon: Icon,
		...rest
	}: StandardProps & {
		/** The attribute shown as the value. */
		attrID: string;
		/** Row title in the popup, the headline-cased attribute ID by default. */
		label?: string;
		icon: Component;
	} = $props();

	const attr = $derived(findAttr(service, attrID));
	const text = $derived(attr === undefined ? '' : formatValue(attr.value, attr.spec));

	// svelte-ignore state_referenced_locally
	const serviceAction = new ServiceAction(deviceID, endpointID, service.id);
</script>

{#snippet icon()}
	<Icon />
{/snippet}

{#snippet details()}
	{#if text !== ''}
		<p>{text}</p>
	{/if}
{/snippet}

<ServiceRoot {deviceID} {endpointID} {...rest} {service} {icon} {details}>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if attr !== undefined}
			<div>{label ?? toHeadlineCase(attrID)}</div>
			<div class="col-span-2">{text === '' ? '-' : text}</div>
		{/if}
	</div>
	<OthersContent {deviceID} {endpointID} {service} action={serviceAction} exclude={[attrID]} />
</ServiceRoot>
