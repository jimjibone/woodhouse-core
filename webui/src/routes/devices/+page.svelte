<script lang="ts">
	import { onDestroy } from 'svelte';
	import { DevicesStore, type DevicesStoreType } from '$lib/stores/devices-stream';
	import { ServiceEnumerator } from '$lib/components/wh/service';
	import TimeSince from '$lib/components/wh/ui/time-since.svelte';
	import { attributeToDate } from '$lib/tools/time';
	import { cn } from "$lib/utils";
	import { BatteryWarningIcon, BatteryLowIcon, BatteryMediumIcon, BatteryFullIcon } from '@lucide/svelte';

	let store: DevicesStoreType;
	const unsubscribe = DevicesStore.subscribe((val) => store = val);
	onDestroy(unsubscribe);
</script>

<main class="grid gap-4 md:grid-cols-1 lg:grid-cols-2 mb-20 md:mb-0">
	{#each store.devices as dev, i (dev.id)}
		{@const deviceName = dev.name ? dev.name : dev.id}
		<div class={cn('rounded-lg border bg-card/50 p-2 text-card-foreground shadow-sm text-left overflow-clip', !dev.online && 'bg-muted/80')}>
			<div class="flex flex-col gap-2">
				<div class="grid grid-cols-[1fr_auto] gap-1 max-w-full">
					<a class={cn("font-semibold", !dev.name && "font-mono")} href={"/devices/"+dev.id}>
						{#if dev.name}
							{dev.name}
						{:else}
							{dev.id}
						{/if}
					</a>
					<span class="flex flex-row gap-2 text-sm items-center whitespace-pre">
						{#if dev.lastSeen}
							<TimeSince past={attributeToDate(dev.lastSeen)}/>
						{/if}
						{#if dev.batteryLevel}
							<!-- <span class={cn("shrink flex flex-row gap-1 text-sm items-center text-muted-foreground", batteryLevel < 33 && "text-warning-foreground", batteryLevel < 20 && "text-error-foreground")}> -->
							<span class={cn("flex flex-row gap-0 text-muted-foreground", dev.batteryLevel < 33 && "text-warning-foreground", dev.batteryLevel < 20 && "text-error-foreground")}>
								{#if dev.batteryLevel < 20}
									<BatteryWarningIcon class="size-5"/>
								{:else if dev.batteryLevel < 33}
									<BatteryLowIcon class="size-5"/>
								{:else if dev.batteryLevel < 66}
									<BatteryMediumIcon class="size-5"/>
								{:else}
									<BatteryFullIcon class="size-5"/>
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

<div class="pt-4">
	<p>Connected: {store.connected}</p>
	<p>Backoff: {store.backoff}</p>
	<p>Devices: {store.devices.length}</p>
</div>
