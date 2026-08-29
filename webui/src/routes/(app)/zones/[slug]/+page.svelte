<script lang="ts">
	import { page } from '$app/state';
	import { DevicesStore, type DevicesStoreDevice } from '$lib/stores/devices-stream';
	import { ZonesStore } from '$lib/stores/zones-stream';
	import type { Zone } from '$lib/api/v1/clients/zone_pb';
	import { DeviceCard } from '$lib/components/wh/device';
	import { zoneIcon } from '$lib/zone-icons';
	import { MapPinIcon } from '@lucide/svelte';
	import { search } from '$lib/stores/search';
	import Fuse from 'fuse.js';
	import { onDestroy } from 'svelte';
	import { useConnectionContext } from '$lib/stores/connection-status.svelte';

	let devices = $state<DevicesStoreDevice[]>([]);
	let zones = $state<Zone[]>([]);
	let zonesConnected = $state(false);
	let query = $state('');

	const connStatus = useConnectionContext();

	onDestroy(
		DevicesStore.subscribe((update) => {
			devices = update.devices;
			connStatus.set(update.connected, !update.connected && update.backoff > 0);
		})
	);
	onDestroy(
		ZonesStore.subscribe((update) => {
			zones = update.zones;
			zonesConnected = update.connected;
		})
	);
	onDestroy(search.subscribe((update) => (query = update.query)));
	onDestroy(() => connStatus.reset());

	const zoneID = $derived(page.params.slug);
	const zone = $derived(zones.find((z) => z.id === zoneID));

	// Only call a zone missing once the stream has actually delivered its state,
	// otherwise a reload shows "not found" for a moment before the zone arrives.
	const zoneMissing = $derived(zonesConnected && zone === undefined);

	// Zone membership is a list of device IDs; the devices themselves come from
	// the devices stream, so join the two here. Ordering follows the devices
	// stream so it matches the Devices page.
	const zoneDevices = $derived.by(() => {
		if (!zone) return [];
		const members = new Set(zone.deviceIds);
		return devices.filter((d) => members.has(d.id));
	});

	let fuse: Fuse<DevicesStoreDevice> | null = $state(null);

	$effect(() => {
		fuse = new Fuse(zoneDevices, {
			threshold: 0.3,
			includeScore: true,
			keys: ['name']
		});
	});

	const filtered = $derived.by(() => {
		if (!fuse) return zoneDevices;
		if (!query.trim()) return zoneDevices;

		return fuse.search(query).map((r) => r.item);
	});
</script>

{#if zoneMissing}
	<main class="flex flex-col items-center justify-center min-h-[65vh] gap-6 text-center px-4">
		<div class="rounded-full bg-muted p-8">
			<MapPinIcon class="size-12 text-muted-foreground" strokeWidth={1.5} />
		</div>
		<div class="grid gap-2 max-w-xs">
			<h2 class="text-xl font-semibold">Zone not found</h2>
			<p class="text-sm text-muted-foreground leading-relaxed">
				This zone no longer exists. See
				<a href="/zones" class="text-foreground underline underline-offset-2 hover:no-underline">all zones</a>.
			</p>
		</div>
	</main>
{:else if zone}
	{@const ZoneIcon = zoneIcon(zone.icon)}
	<main class="flex flex-col gap-3 mb-20 md:mb-0">
		<div class="flex items-center gap-2">
			<ZoneIcon class="size-5 text-muted-foreground" />
			<h1 class="text-lg font-semibold">{zone.name}</h1>
		</div>

		{#if zoneDevices.length === 0}
			<div class="flex flex-col items-center gap-2 py-12 text-center">
				<p class="text-sm text-muted-foreground">
					No devices in this zone yet. Assign some under
					<a href="/settings/zones" class="text-foreground underline underline-offset-2 hover:no-underline">
						Settings &rsaquo; Zones
					</a>.
				</p>
			</div>
		{:else}
			<div class="grid gap-2 md:grid-cols-1 lg:grid-cols-2">
				{#each filtered as dev (dev.id)}
					<DeviceCard device={dev} />
				{:else}
					<div class="col-span-full flex flex-col items-center gap-2 py-12 text-center">
						<p class="text-sm text-muted-foreground">
							No results for "<span class="font-medium text-foreground">{query}</span>"
						</p>
					</div>
				{/each}
			</div>
		{/if}
	</main>
{/if}
