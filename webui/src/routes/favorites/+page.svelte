<script lang="ts">
	import { onDestroy } from 'svelte';
	import { FavoritesStore, type FavoritesStoreType } from '$lib/stores/favorites-stream';
	import { ServiceEnumerator } from '$lib/components/wh/service';
	import { valueToDate } from '$lib/tools/time';

	let store: FavoritesStoreType;
	const unsubscribe = FavoritesStore.subscribe((val: FavoritesStoreType) => store = val);
	onDestroy(unsubscribe);
</script>

<main class="grid gap-4 md:grid-cols-2 lg:grid-cols-3 mb-20 md:mb-0">
	{#each store.deviceServices as dev, i (dev.key)}
		{#if dev.service !== undefined}
			<ServiceEnumerator
				deviceName={dev.deviceName ? dev.deviceName : ""}
				deviceID={dev.deviceId}
				online={dev.online ? dev.online : false}
				lastSeen={dev.lastSeen ? valueToDate(dev.lastSeen) : undefined}
				batteryLevel={dev.batteryLevel}
				service={dev.service}/>
		{/if}
	{:else}
		<div>
			<p>No favorites!</p>
		</div>
	{/each}
</main>

<div class="pt-4">
	<p>Connected: {store.connected}</p>
	<p>Backoff: {store.backoff}</p>
	<p>Services: {store.deviceServices.length}</p>
</div>
