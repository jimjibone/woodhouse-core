<script lang="ts">
	import { onDestroy } from 'svelte';
	import { DevicesStore, type DevicesStoreType } from '$lib/stores/devices-stream';
	import { ServiceRoot, ServiceEnumerator } from '$lib/components/wh/service';
	import { Service_ServiceType } from '$lib/api/v1/clients/client_service_pb';
	import { cn } from "$lib/utils";

	let store: DevicesStoreType;
	const unsubscribe = DevicesStore.subscribe((val) => store = val);
	onDestroy(unsubscribe);
</script>

<h1>Welcome to Devices!</h1>
<p>Connected: {store.connected}</p>
<p>Backoff: {store.backoff}</p>
<p>Devices: {store.devices.length}</p>

<main class="grid gap-4 md:p-4 md:grid-cols-1 lg:grid-cols-2 mb-20 md:mb-0">
	{#each store.devices as dev, i (dev.id)}
		{@const deviceName = dev.name ? dev.name : dev.id}
		<div class={cn('rounded-lg border bg-card/50 p-2 text-card-foreground shadow-sm text-left overflow-clip', !dev.online && 'bg-muted/80')}>
			<div class="flex flex-col gap-2">
				<div class="flex flex-row items-center">
					{#if dev.name}
						<p class="font-semibold">{dev.name}</p>
					{:else}
						<p class="font-semibold font-mono">{dev.id}</p>
					{/if}
				</div>
				<div class="flex flex-row gap-2 overflow-x-scroll">
					{#each dev.services as srv, i (srv.id)}
						<ServiceEnumerator
							showDeviceName={false}
							deviceName={deviceName}
							deviceID={dev.id}
							online={dev.online}
							service={srv} />
					{/each}
				</div>
			</div>
		</div>
	{:else}
		<div>
			<p>No devices!</p>
		</div>
	{/each}
</main>
