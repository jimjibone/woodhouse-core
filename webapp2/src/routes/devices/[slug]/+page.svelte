<script lang="ts">
	import { page } from '$app/stores';
	import { Device } from '@/api/v1/clients/client_service_pb';
	import { onDestroy } from 'svelte';
	import { DeviceStore, type DeviceStoreType, DeviceAction } from '$lib/stores';
	import DeviceComponent from '../Device.svelte';

	const deviceId = $page.params.slug;
	let connected = false;
	let backoff = 0;
	let device: Device | undefined = undefined;
	const unsubscribe = DeviceStore.subscribe((val: DeviceStoreType) => {
		connected = val.connected;
		backoff = val.backoff;
		for (let d = 0; d < val.devices.length; d++) {
			if (val.devices[d].id === deviceId) {
				// console.log("streamDevices: update: found " + response.id);
				device = val.devices[d];
				break;
			}
		}
	});
	onDestroy(unsubscribe);
</script>

<header class="bg-background sticky top-0 z-10 flex h-[57px] items-center gap-1 border-b px-4">
	<h1 class="text-xl font-semibold">Device: {deviceId}{connected ? "" : " - Disconnected (backoff=" + backoff + "ms)"}</h1>
</header>
<main class="grid flex-1 gap-4 overflow-auto p-4">
	<div class="relative flex gap-4 h-full min-h-[50vh] flex-col rounded-xl">
	{#if device !== undefined}
		<DeviceComponent device={device} onAction={DeviceAction} />
	{:else}
		<p>No device found</p>
	{/if}
	</div>
</main>
