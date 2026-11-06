<script lang="ts">
	// Renders any service from its specs: each attribute by its spec kind,
	// each command as a button or a small form. Used for the generic service
	// and for every service type that has no typed component.
	import type { Attribute } from '$lib/api/v2/clients/client_service_pb';
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { SlidersHorizontalIcon } from '@lucide/svelte';
	import { OthersContent } from '$lib/components/wh/attributes';
	import { formatValue, isWritable, specKind, valueKind } from '$lib/device';

	let { deviceID, endpointID, service, ...rest }: StandardProps = $props();

	let serviceAction = new ServiceAction(deviceID, endpointID, service.id);

	// Writable controls first, then read-only values, each group ordered by ID.
	let attrs = $derived(
		[...service.attrs].sort((a, b) => {
			const aw = isWritable(a) ? 0 : 1;
			const bw = isWritable(b) ? 0 : 1;
			if (aw !== bw) return aw - bw;
			return a.id < b.id ? -1 : a.id > b.id ? 1 : 0;
		})
	);

	// Only simple values go in the condensed summary line; lists, structs and
	// images are shown in the popup.
	const summarisable = (attr: Attribute) => {
		const kind = specKind(attr.spec) ?? valueKind(attr.value);
		return kind !== undefined && kind !== 'list' && kind !== 'struct' && kind !== 'image';
	};

	const maxSummaryValues = 3;
	let summaryValues = $derived(
		attrs
			.filter(summarisable)
			.map((attr) => formatValue(attr.value, attr.spec))
			.filter((val) => val !== '')
	);
	let shownValues = $derived(summaryValues.slice(0, maxSummaryValues));
	let moreCount = $derived(summaryValues.length - shownValues.length);
</script>

{#snippet icon()}
	<SlidersHorizontalIcon />
{/snippet}

{#snippet details()}
	{#each shownValues as val, i}
		<p class={i > 0 ? 'text-muted-foreground' : undefined}>{val}</p>
	{/each}
	{#if moreCount > 0}
		<p class="text-muted-foreground">+{moreCount}</p>
	{/if}
{/snippet}

<ServiceRoot
	{deviceID}
	{endpointID}
	{...rest}
	{service}
	fallbackAlias={service.id}
	actionPending={serviceAction.pending}
	errorSignal={serviceAction.error}
	{icon}
	{details}
>
	<OthersContent {deviceID} {endpointID} {service} action={serviceAction} separator={false} />
</ServiceRoot>
