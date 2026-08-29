<script lang="ts">
	import type { DevicesStoreDevice } from '$lib/stores/devices-stream';
	import { ServiceEnumerator } from '$lib/components/wh/service';
	import TimeSince from '$lib/components/wh/ui/time-since.svelte';
	import { attributeToDate } from '$lib/tools/time';
	import { cn } from '$lib/utils';
	import { Device_DeviceType } from '$lib/api/v1/clients/client_service_pb';
	import {
		BatteryWarningIcon,
		BatteryLowIcon,
		BatteryMediumIcon,
		BatteryFullIcon,
		LayersIcon
	} from '@lucide/svelte';

	let { device: dev }: { device: DevicesStoreDevice } = $props();

	const deviceName = $derived(dev.name ? dev.name : dev.id);
</script>

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
				{#if dev.typ === Device_DeviceType.GROUPED}
					<LayersIcon class="inline-block size-4 mb-0.5 text-muted-foreground" role="img" aria-label="Group" />
				{/if}
			</a>
			<span class="flex flex-row gap-2 text-sm md:text-xs items-center whitespace-pre">
				{#if dev.lastSeen}
					<TimeSince past={attributeToDate(dev.lastSeen)} />
				{/if}
				{#if dev.batteryLevel}
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
			{#each dev.services as srv (srv.id)}
				<ServiceEnumerator
					showDeviceName={false}
					naturalWidth
					{deviceName}
					deviceID={dev.id}
					deviceType={dev.typ}
					online={dev.online}
					service={srv}
				/>
			{/each}
		</div>
	</div>
</div>
