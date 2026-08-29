<script lang="ts">
	import { ZonesStore as store } from '$lib/stores/zones-stream';
	import { DevicesStore } from '$lib/stores/devices-stream';
	import type { Zone } from '$lib/api/v1/clients/zone_pb';
	import type { DevicesStoreDevice } from '$lib/stores/devices-stream';
	import { zoneIcon } from '$lib/zone-icons';
	import { MapPinIcon, TriangleAlertIcon } from '@lucide/svelte';
	import { onDestroy } from 'svelte';
	import { useConnectionContext } from '$lib/stores/connection-status.svelte';
	import { userData } from '$lib/stores/auth-store';
	import { toSentenceCaseLocalized } from '$lib/tools/headline-case';

	let zones = $state<Zone[]>([]);
	let devices = $state<DevicesStoreDevice[]>([]);
	let streamError = $state<string | null>(null);

	const connStatus = useConnectionContext();
	onDestroy(
		store.subscribe((update) => {
			zones = update.zones;
			streamError = update.error;
			connStatus.set(update.connected, !update.connected && update.backoff > 0);
		})
	);
	onDestroy(DevicesStore.subscribe((update) => (devices = update.devices)));
	onDestroy(() => connStatus.reset());

	const knownIDs = $derived(new Set(devices.map((d) => d.id)));

	// A zone can name a device the devices stream has not sent yet, so count
	// what is actually there rather than the raw membership length.
	function presentCount(zone: Zone): number {
		return zone.deviceIds.filter((id) => knownIDs.has(id)).length;
	}

	const isAdmin = $derived($userData.role === 'admin');
</script>

{#if streamError}
	<div
		class="mb-4 flex items-start gap-3 rounded-md border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive"
	>
		<TriangleAlertIcon class="mt-0.5 size-4 shrink-0" />
		<div>
			<p class="font-medium">Unable to load zones</p>
			<p class="mt-0.5 text-destructive/80">{toSentenceCaseLocalized(streamError)}</p>
		</div>
	</div>
{/if}

{#if zones.length === 0}
	<main class="flex flex-col items-center justify-center min-h-[65vh] gap-6 text-center px-4">
		<div class="rounded-full bg-muted p-8">
			<MapPinIcon class="size-12 text-muted-foreground" strokeWidth={1.5} />
		</div>
		<div class="grid gap-2 max-w-xs">
			<h2 class="text-xl font-semibold">No zones yet</h2>
			<p class="text-sm text-muted-foreground leading-relaxed">
				{#if isAdmin}
					Zones group your devices by room or area. Create one under
					<a href="/settings/zones" class="text-foreground underline underline-offset-2 hover:no-underline">
						Settings &rsaquo; Zones
					</a>
					and it will appear in the navigation.
				{:else}
					Zones group your devices by room or area. Ask an admin to set them up.
				{/if}
			</p>
		</div>
	</main>
{:else}
	<main class="grid gap-2 grid-cols-2 md:grid-cols-3 lg:grid-cols-4 mb-20 md:mb-0">
		{#each zones as zone (zone.id)}
			{@const ZoneIcon = zoneIcon(zone.icon)}
			{@const count = presentCount(zone)}
			<a
				href={'/zones/' + zone.id}
				class="flex flex-col items-start gap-3 rounded-xl border bg-card/50 p-4 text-card-foreground shadow-sm hover:bg-accent"
			>
				<ZoneIcon class="size-6 text-muted-foreground" />
				<div class="grid gap-0.5 min-w-0 w-full">
					<span class="truncate font-semibold">{zone.name}</span>
					<span class="text-muted-foreground text-xs">
						{count}
						{count === 1 ? 'device' : 'devices'}
					</span>
				</div>
			</a>
		{/each}
	</main>
{/if}
