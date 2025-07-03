<script lang="ts">
	import { onDestroy } from 'svelte';
	import { FavoritesStore, type FavoritesStoreType } from '$lib/stores/favorites-stream';
	import { ServiceRoot, Climate, Lightbulb } from '$lib/components/wh/service';
	import { Service_ServiceType } from '$lib/api/v1/clients/client_service_pb';

	let store: FavoritesStoreType;
	const unsubscribe = FavoritesStore.subscribe((val: FavoritesStoreType) => store = val);
	onDestroy(unsubscribe);
</script>

<h1>Welcome to Favourites!</h1>
<p>Connected: {store.connected}</p>
<p>Backoff: {store.backoff}</p>
<p>Services: {store.deviceServices.length}</p>

<main class="grid gap-4 md:p-4 md:grid-cols-2 lg:grid-cols-3 mb-20 md:mb-0">
	{#each store.deviceServices as dev, i (dev.key)}
		{#if dev.service !== undefined}
			{#if dev.service?.typ == Service_ServiceType.CLIMATE}
				<Climate
					deviceName={dev.deviceName}
					deviceID={dev.deviceId}
					online={dev.online}
					service={dev.service}/>
			{:else if dev.service?.typ == Service_ServiceType.LIGHTBULB}
				<Lightbulb
					deviceName={dev.deviceName}
					deviceID={dev.deviceId}
					online={dev.online}
					service={dev.service}/>
			{:else}
				<ServiceRoot
					deviceName={dev.deviceName}
					deviceID={dev.deviceId}
					online={dev.online}
					service={dev.service}
					actionPending={false}
					errorSignal={null}>
				</ServiceRoot>
			{/if}
		{/if}
	{:else}
		<div>
			<p>No favorites!</p>
		</div>
	{/each}
</main>
