<script lang="ts">
	import { onMount, onDestroy } from 'svelte/internal';
	import type { Unsubscriber } from 'svelte/store';
	import { devicesStream, DeviceInfoState } from '../stores/devices';
	import DevicePageItem from './DevicePageItem.svelte';

	let devices: DeviceInfoState[] = [];
	let connected: boolean = false;
	let unsubscribeDevices: Unsubscriber = null;
	let unsubscribeConnected: Unsubscriber = null;

	onMount(async () => {
		unsubscribeDevices = devicesStream.subscribeData(value => { devices = value; });
		unsubscribeConnected = devicesStream.subscribeConnected(value => { connected = value; });
	});

	onDestroy(() => {
		unsubscribeDevices();
		unsubscribeConnected();
	});
</script>

<section class="hero">
	<div class="hero-body">
		<p class="title">
			Devices
		</p>
		<p class="subtitle">
			{#if connected}
				Connected - {devices.length} {devices.length == 1 ? "device" : "devices"}
			{:else}
				Disconnected
			{/if}
		</p>
	</div>
</section>

{#each devices as device (device.fullId)}
<DevicePageItem device={device} />
{:else}
<div class="column is-full">
	<div class="box">
		No devices.
	</div>
</div>
{/each}
