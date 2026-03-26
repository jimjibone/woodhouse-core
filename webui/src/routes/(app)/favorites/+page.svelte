<script lang="ts">
	import { FavoritesStore as store } from '$lib/stores/favorites-stream';
	import { type DeviceService } from '$lib/api/v1/clients/user_service_pb';
	import { ServiceEnumerator } from '$lib/components/wh/service';
	import { valueToDate } from '$lib/tools/time';
	import { search } from '$lib/stores/search';
	import Fuse from 'fuse.js';
	import { onDestroy } from 'svelte';
	import { useConnectionContext } from '$lib/stores/connection-status.svelte';
	import { HeartIcon } from '@lucide/svelte';

	let services = $state<DeviceService[]>([]);
	let query = $state('');

	const connStatus = useConnectionContext();

	onDestroy(
		store.subscribe((update) => {
			services = update.deviceServices;
			connStatus.set(update.connected, !update.connected && update.backoff > 0);
		})
	);
	onDestroy(search.subscribe((update) => (query = update.query)));
	onDestroy(() => connStatus.reset());

	let filtered = $derived.by(() => {
		if (!fuse) return services;
		if (!query.trim()) return services;

		return fuse.search(query).map((r) => r.item);
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

{#if services.length === 0}
	<main class="flex flex-col items-center justify-center min-h-[65vh] gap-6 text-center px-4">
		<div class="relative">
			<div class="rounded-full bg-muted p-8">
				<HeartIcon class="size-12 text-muted-foreground" strokeWidth={1.5} />
			</div>
		</div>
		<div class="grid gap-2 max-w-xs">
			<h2 class="text-xl font-semibold">No favorites yet</h2>
			<p class="text-sm text-muted-foreground leading-relaxed">
				Favorite a service from the
				<a href="/devices" class="text-foreground underline underline-offset-2 hover:no-underline"> Devices </a>
				page to pin it here for quick access.
			</p>
		</div>
	</main>
{:else}
	<main class="grid gap-2 md:grid-cols-2 lg:grid-cols-3 mb-20 md:mb-0">
		{#each filtered as dev (dev.key)}
			{#if dev.service !== undefined}
				<ServiceEnumerator
					deviceName={dev.deviceName ? dev.deviceName : ''}
					deviceID={dev.deviceId}
					online={dev.online ? dev.online : false}
					lastSeen={dev.lastSeen ? valueToDate(dev.lastSeen) : undefined}
					batteryLevel={dev.batteryLevel}
					service={dev.service}
				/>
			{/if}
		{:else}
			<div class="col-span-full flex flex-col items-center gap-2 py-12 text-center">
				<p class="text-sm text-muted-foreground">
					No results for "<span class="font-medium text-foreground">{query}</span>"
				</p>
			</div>
		{/each}
	</main>
{/if}
