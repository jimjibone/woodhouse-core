<script lang="ts">
	// A device's recent events, newest first. Replayed events (those sent from
	// the server's buffer on connect) look the same as live ones.
	import type { DevicesStoreDevice } from '$lib/stores/devices-stream';
	import type { Event, FieldSpec } from '$lib/api/v2/clients/client_service_pb';
	import { endpointLabel, findEndpoint, findService, formatValue, timeToDate } from '$lib/device';
	import { toHeadlineCase } from '$lib/tools/headline-case';
	import TimeSince from '$lib/components/wh/ui/time-since.svelte';
	import ValueView from '$lib/components/wh/attributes/value-view.svelte';

	let {
		device,
		eventsLost = false
	}: {
		device: DevicesStoreDevice;
		/** Some events before the ones shown are no longer available. */
		eventsLost?: boolean;
	} = $props();

	const endpointName = (ev: Event): string => {
		const ep = findEndpoint(device, ev.endpointId);
		const label = ep ? endpointLabel(ep) : `Endpoint ${ev.endpointId}`;
		return label === '' ? 'Device' : label;
	};

	const serviceName = (ev: Event): string => {
		const srv = findService(device, ev.endpointId, ev.serviceId);
		return toHeadlineCase(srv?.alias ? srv.alias : ev.serviceId);
	};

	const fieldSpecs = (ev: Event): FieldSpec[] =>
		findService(device, ev.endpointId, ev.serviceId)?.events.find((e) => e.id === ev.id)?.fields ?? [];

	const eventDate = (ev: Event): Date | undefined => (ev.time ? timeToDate(ev.time) : undefined);

	// Scalars read better on one line; lists and structs get the table view.
	const isSimple = (kind: string | undefined) => kind !== 'list' && kind !== 'struct';
</script>

<div class="rounded-xl border bg-card/50 shadow-sm">
	{#if device.events.length === 0}
		<p class="p-4 text-sm text-muted-foreground">No events yet.</p>
	{:else}
		<ul class="divide-y">
			{#each device.events as ev (ev.epoch + ':' + ev.seq)}
				{@const specs = fieldSpecs(ev)}
				{@const when = eventDate(ev)}
				<li class="grid grid-cols-[1fr_auto] gap-x-3 gap-y-0.5 px-4 py-2 text-sm">
					<div class="min-w-0">
						<span class="font-medium">{toHeadlineCase(ev.id)}</span>
						<span class="text-muted-foreground">
							{endpointName(ev)} / {serviceName(ev)}
						</span>
					</div>
					<div class="text-xs">
						{#if when}
							<TimeSince past={when} warn={false} />
						{/if}
					</div>
					{#if ev.fields.length > 0}
						<div class="col-span-2 grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5 text-xs">
							{#each ev.fields as f (f.id)}
								{@const spec = specs.find((s) => s.id === f.id)?.spec}
								<span class="text-muted-foreground">{toHeadlineCase(f.id)}</span>
								{#if isSimple(f.value?.kind.case)}
									<span class="break-words">{formatValue(f.value, spec)}</span>
								{:else}
									<div class="min-w-0"><ValueView value={f.value} {spec} /></div>
								{/if}
							{/each}
						</div>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
	{#if eventsLost}
		<p class="border-t px-4 py-2 text-xs text-muted-foreground">Some earlier events are no longer available.</p>
	{/if}
</div>
