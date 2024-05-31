<script lang="ts">
	import { onDestroy } from 'svelte';
	import { DeviceStore, type DeviceStoreType, DeviceAction } from '$lib/stores';
	import DeviceComponent from './Device.svelte';

	let store: DeviceStoreType;
	const unsubscribe = DeviceStore.subscribe((val: DeviceStoreType) => store = val);
	onDestroy(unsubscribe);
</script>

<header class="bg-background sticky top-0 z-10 flex h-[57px] items-center gap-1 border-b px-4">
	<h1 class="text-xl font-semibold">Devices{store.connected ? "" : " - Disconnected (backoff=" + store.backoff + "ms)"}</h1>
</header>
<main class="grid flex-1 gap-4 overflow-auto p-4">
	<div class="relative flex gap-4 h-full min-h-[50vh] flex-col rounded-xl">
		{#each store.devices as dev, i}
		<DeviceComponent device={dev} onAction={DeviceAction}/>
		{/each}
	</div>
</main>
