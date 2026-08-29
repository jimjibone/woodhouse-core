<script lang="ts">
	import { DevicesStore as store, type DevicesStoreDevice } from '$lib/stores/devices-stream';
	import { DeviceCard } from '$lib/components/wh/device';
	import { LampIcon } from '@lucide/svelte';
	import { search } from '$lib/stores/search';
	import Fuse from 'fuse.js';
	import { onDestroy } from 'svelte';
	import { useConnectionContext } from '$lib/stores/connection-status.svelte';

	let devices = $state<DevicesStoreDevice[]>([]);
	let query = $state('');

	const connStatus = useConnectionContext();

	onDestroy(
		store.subscribe((update) => {
			devices = update.devices;
			connStatus.set(update.connected, !update.connected && update.backoff > 0);
		})
	);
	onDestroy(search.subscribe((update) => (query = update.query)));
	onDestroy(() => connStatus.reset());

	let filtered = $derived.by(() => {
		if (!fuse) return devices;
		if (!query.trim()) return devices;

		return fuse.search(query).map((r) => r.item);
	});

	let fuse: Fuse<DevicesStoreDevice> | null = $state(null);

	// Reactively rebuild Fuse whenever `devices` changes.
	$effect(() => {
		fuse = new Fuse(devices, {
			threshold: 0.3,
			includeScore: true,
			keys: ['name']
		});
	});
</script>

{#if devices.length === 0}
	<main class="flex flex-col items-center justify-center min-h-[65vh] gap-6 text-center px-4">
		<div class="rounded-full bg-muted p-8">
			<LampIcon class="size-12 text-muted-foreground" strokeWidth={1.5} />
		</div>
		<div class="grid gap-2 max-w-xs">
			<h2 class="text-xl font-semibold">No devices yet</h2>
			<p class="text-sm text-muted-foreground leading-relaxed">
				Once a client pairs and registers its devices they will appear here. Check
				<a href="/settings/clients" class="text-foreground underline underline-offset-2 hover:no-underline"> Clients </a>
				for more details on connected clients and pairing new ones.
			</p>
		</div>
	</main>
{:else}
	<main class="grid gap-2 md:grid-cols-1 lg:grid-cols-2 mb-20 md:mb-0">
		{#each filtered as dev (dev.id)}
			<DeviceCard device={dev} />
		{:else}
			<div class="col-span-full flex flex-col items-center gap-2 py-12 text-center">
				<p class="text-sm text-muted-foreground">
					No results for "<span class="font-medium text-foreground">{query}</span>"
				</p>
			</div>
		{/each}
	</main>
{/if}
