<script lang="ts">
	import { FavoritesStore as store } from '$lib/stores/favorites-stream';
	import { type DeviceService } from '$lib/api/v1/clients/user_service_pb';
	import { ServiceEnumerator } from '$lib/components/wh/service';
	import { valueToDate } from '$lib/tools/time';
	import { search } from '$lib/stores/search';
	import Fuse from 'fuse.js';
	import { onDestroy } from 'svelte';

	let services = $state<DeviceService[]>([]);
	let query = $state("");
	onDestroy(store.subscribe((update) => services = update.deviceServices));
	onDestroy(search.subscribe((update) => query = update.query));

	let filtered = $derived.by(() => {
		if (!fuse) return services;
		if (!query.trim()) return services;

		return fuse.search(query).map(r => r.item);
	});

	let fuse: Fuse<DeviceService> | null = $state(null);

	// Reactively rebuild Fuse whenever `services` changes.
	$effect(() => {
		fuse = new Fuse(services, {
			threshold: 0.3,
			includeScore: true,
			keys: ['deviceName']
		});
	});
</script>

<main class="grid gap-4 md:grid-cols-2 lg:grid-cols-3 mb-20 md:mb-0">
	{#each filtered as dev, i (dev.key)}
		{#if dev.service !== undefined}
			<ServiceEnumerator
				deviceName={dev.deviceName ? dev.deviceName : ""}
				deviceID={dev.deviceId}
				online={dev.online ? dev.online : false}
				lastSeen={dev.lastSeen ? valueToDate(dev.lastSeen) : undefined}
				batteryLevel={dev.batteryLevel}
				service={dev.service}
			/>
		{/if}
	{:else}
		<div>
			<p>No favorites!</p>
		</div>
	{/each}
</main>

<div class="pt-4">
	<p>Connected: {$store.connected}</p>
	<p>Backoff: {$store.backoff}</p>
	<p>Services: {$store.deviceServices.length}</p>
</div>
