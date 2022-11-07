<script lang="ts">
	import { onMount, onDestroy } from 'svelte/internal';
    import type { Unsubscriber } from 'svelte/store';
	import type { DeviceInfo } from '../api/device_pb';
	import { deviceInfosStream } from '../stores/devices';

	let deviceInfos: DeviceInfo[] = [];
	let deviceInfosConnected: boolean = false;
	let unsubscribeDeviceInfos: Unsubscriber = null;
	let unsubscribeDeviceInfosConnected: Unsubscriber = null;

	onMount(async () => {
		console.log("DevicesPage onMount");
		unsubscribeDeviceInfos = deviceInfosStream.subscribeData(value => { deviceInfos = value; });
		unsubscribeDeviceInfosConnected = deviceInfosStream.subscribeConnected(value => { deviceInfosConnected = value; });
	});

	onDestroy(() => {
		console.log("DevicesPage onDestroy");
		unsubscribeDeviceInfos();
		unsubscribeDeviceInfosConnected();
	});
</script>

<section class="hero">
	<div class="hero-body">
		<p class="title">
			Devices
		</p>
		<p class="subtitle">
			{#if deviceInfosConnected}
				Connected - {deviceInfos.length} {deviceInfos.length == 1 ? "device" : "devices"}
			{:else}
				Disconnected
			{/if}
		</p>
	</div>
</section>
{#each deviceInfos as info (info.getDeviceId())}
<div class="column is-full">
	<div class="box">
		{info.getBridgeId()} {info.getDeviceId()} {info.getName()} <a href="{info.getUrl()}">{info.getUrl()}</a>
	</div>
</div>
{:else}
<div class="column is-full">
	<div class="box">
		No devices.
	</div>
</div>
{/each}
