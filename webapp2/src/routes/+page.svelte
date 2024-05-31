<script lang="ts">
	import { onDestroy } from 'svelte';
	import { DeviceAction, DeviceStore, type DeviceStoreType } from '$lib/stores';
	import Service from './devices/Service.svelte';
	import { getDeviceInfo, getDeviceName } from '$lib/apitools';

	let store: DeviceStoreType;
	const unsubscribe = DeviceStore.subscribe((val: DeviceStoreType) => store = val);
	onDestroy(unsubscribe);
</script>

<header class="bg-background sticky top-0 z-10 flex h-[57px] items-center gap-1 border-b px-4">
	<h1 class="text-xl font-semibold">Dashboard{store.connected ? "" : " - Disconnected (backoff=" + store.backoff + "ms)"}</h1>
</header>
<main class="grid gap-4 p-4 md:grid-cols-2 lg:grid-cols-3">
	{#each store.devices as dev, i (dev.id)}
		{#each dev.services as srv, i (srv.id)}
			{@const info = getDeviceInfo(dev)}
			<Service title={info.name} online={info.online} service={srv} onAction={(serviceID, val) => {
				return DeviceAction(dev.id, serviceID, val);
			}}/>
		{/each}
	{:else}
		<div>
			<p>No devices!</p>
		</div>
	{/each}
</main>
