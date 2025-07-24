<script lang="ts">
	import { onDestroy } from 'svelte';
	import { page } from '$app/state';
	import { DevicesStore, type DevicesStoreDevice } from '$lib/stores/devices-stream';
	import { ServiceEnumerator } from '$lib/components/wh/service';
	import { ServiceSchema } from '$lib/api/v1/clients/client_service_pb';
	import { toJsonString } from "@bufbuild/protobuf";
	import { attributeToDate, toHumanDate } from '$lib/tools/time';

	const deviceID = page.params.slug;

	let connected: boolean;
	let backoff: number;
	let dev: DevicesStoreDevice | undefined;
	const unsubscribe = DevicesStore.subscribe((store) => {
		connected = store.connected;
		backoff = store.backoff;
		for (const it of store.devices) {
			if (it.id === deviceID) {
				dev = it;
				break;
			}
		}
	});
	onDestroy(unsubscribe);
</script>

<main class="">
	{#if dev}
		{@const deviceName = dev.name ? dev.name : dev.id}
		<div class="flex flex-col gap-2">
			<div class="flex flex-row items-center text-xl">
				{#if dev.name}
					<p class="font-semibold">{dev.name}</p>
				{:else}
					<p class="font-semibold font-mono">{dev.id}</p>
				{/if}
			</div>

			<div class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 items-center">
				<div>Device Name</div><div class="font-mono bg-muted p-1 rounded-md">{dev.name}</div>
				<div>Device ID</div><div class="font-mono bg-muted p-1 rounded-md">{dev.id}</div>
				<div>Online</div><div class="font-mono bg-muted p-1 rounded-md">{dev.online}</div>
				{#if dev.lastSeen}
					<div>Last Seen</div><div class="font-mono bg-muted p-1 rounded-md">{toHumanDate(attributeToDate(dev.lastSeen))}</div>
				{/if}
				{#if dev.batteryLevel}
					<div>Battery</div><div class="font-mono bg-muted p-1 rounded-md">{Number(dev.batteryLevel)}%</div>
				{/if}
				<div class="col-span-2">Services:</div>
			</div>

			<div class="flex flex-col md:flex-row gap-2 overflow-x-scroll">
				{#each dev.services as srv, i (srv.id)}
					<ServiceEnumerator
						showDeviceName={false}
						deviceName={deviceName}
						deviceID={dev.id}
						online={dev.online}
						service={srv} />
				{/each}
			</div>

			{#each dev.services as srv, i (srv.id)}
				<div class="min-w-0 overflow-x-scroll font-mono bg-muted px-4 py-2 rounded-md whitespace-pre text-sm">
					{toJsonString(ServiceSchema, srv, {prettySpaces: 2})}
				</div>
			{/each}
		</div>
	{:else}
		<div class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 items-center">
			<div class="text-xl col-span-2">Device not found</div>
			<div>Device ID</div><div class="font-mono bg-muted p-1 rounded-md">{deviceID}</div>
			<div>Server:</div><div class="font-mono bg-muted p-1 rounded-md">{connected ? "Connected" : "Disconnected (backoff="+backoff+"ms"}</div>
			<!-- <div>Backoff:</div><div class="font-mono bg-muted p-1 rounded-md">{backoff}</div> -->
		</div>
	{/if}
</main>
