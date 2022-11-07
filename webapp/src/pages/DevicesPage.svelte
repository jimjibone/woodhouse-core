<script lang="ts">
	import { onMount, onDestroy } from 'svelte/internal';
	import type { Unsubscriber } from 'svelte/store';
	import type { DeviceInfo, DeviceState } from '../api/device_pb';
    import DeviceValue from '../components/DeviceValue.svelte';
	import { devicesStream, DeviceInfoState } from '../stores/devices';

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
<div class="column is-full">
	<div class="box">
		{device.fullId}
		{#if device.info != null}
			{device.info.getBridgeId()} {device.info.getDeviceId()} {device.info.getName()} <a href="{device.info.getUrl()}">{device.info.getUrl()}</a>
		{/if}
		{#if device.state != null}
			{device.state.getBridgeId()} {device.state.getDeviceId()} {device.state.getValuesList().length}
			{#each device.state.getValuesList() as value (value.getName())}
				<DeviceValue value={value} />
			{/each}
		{/if}
	</div>
</div>
{:else}
<div class="column is-full">
	<div class="box">
		No devices.
	</div>
</div>
{/each}
