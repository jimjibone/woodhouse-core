<script lang="ts">
	import { DevicesStore as store, type DevicesStoreDevice } from '$lib/stores/devices-stream';
	import { ServiceEnumerator } from '$lib/components/wh/service';
	import TimeSince from '$lib/components/wh/ui/time-since.svelte';
	import { attributeToDate } from '$lib/tools/time';
	import { cn } from '$lib/utils';
	import { BatteryWarningIcon, BatteryLowIcon, BatteryMediumIcon, BatteryFullIcon, LampIcon } from '@lucide/svelte';
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
				<a href="/clients" class="text-foreground underline underline-offset-2 hover:no-underline"> Clients </a>
				for more details on connected clients and pairing new ones.
			</p>
		</div>
	</main>
{:else}
	<main class="grid gap-2 md:grid-cols-1 lg:grid-cols-2 mb-20 md:mb-0">
		{#each filtered as dev, i (dev.id)}
			{@const deviceName = dev.name ? dev.name : dev.id}
			<div
				class={cn(
					'rounded-xl border bg-card/50 p-2 text-card-foreground shadow-sm text-left text-base md:text-sm overflow-clip',
					!dev.online && 'bg-muted/80'
				)}
			>
				<div class="flex flex-col gap-2">
					<div class="grid grid-cols-[1fr_auto] gap-1 max-w-full">
						<a class={cn('font-semibold', !dev.name && 'font-mono')} href={'/devices/' + dev.id}>
							{#if dev.name}
								{dev.name}
							{:else}
								{dev.id}
							{/if}
						</a>
						<span class="flex flex-row gap-2 text-sm md:text-xs items-center whitespace-pre">
							{#if dev.lastSeen}
								<TimeSince past={attributeToDate(dev.lastSeen)} />
							{/if}
							{#if dev.batteryLevel}
								<!-- <span class={cn("shrink flex flex-row gap-1 text-sm items-center text-muted-foreground", batteryLevel < 33 && "text-warning-foreground", batteryLevel < 20 && "text-error-foreground")}> -->
								<span
									class={cn(
										'flex flex-row gap-0 text-muted-foreground',
										dev.batteryLevel < 33 && 'text-warning-foreground',
										dev.batteryLevel < 20 && 'text-error-foreground'
									)}
								>
									{#if dev.batteryLevel < 20}
										<BatteryWarningIcon class="size-5 md:size-4" />
									{:else if dev.batteryLevel < 33}
										<BatteryLowIcon class="size-5 md:size-4" />
									{:else if dev.batteryLevel < 66}
										<BatteryMediumIcon class="size-5 md:size-4" />
									{:else}
										<BatteryFullIcon class="size-5 md:size-4" />
									{/if}
									{Number(dev.batteryLevel)}%
								</span>
							{/if}
						</span>
					</div>
					<div class="flex flex-row gap-2 overflow-x-scroll">
						{#each dev.services as srv, i (srv.id)}
							<ServiceEnumerator
								showDeviceName={false}
								naturalWidth
								{deviceName}
								deviceID={dev.id}
								online={dev.online}
								service={srv}
							/>
						{/each}
					</div>
				</div>
			</div>
		{:else}
			<div class="col-span-full flex flex-col items-center gap-2 py-12 text-center">
				<p class="text-sm text-muted-foreground">
					No results for "<span class="font-medium text-foreground">{query}</span>"
				</p>
			</div>
		{/each}
	</main>
{/if}
