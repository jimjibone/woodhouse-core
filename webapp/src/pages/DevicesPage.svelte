<script lang="ts">
	import { onMount, onDestroy } from 'svelte/internal';
	import type { Unsubscriber } from 'svelte/store';
	import type { DeviceInfo, DeviceState } from '../api/device_pb';
    import DeviceValue from '../components/DeviceValue.svelte';
	import { deviceInfosStream, deviceStatesStream } from '../stores/devices';

	let deviceInfos: DeviceInfo[] = [];
	let deviceInfosConnected: boolean = false;
	let deviceStates: DeviceState[] = [];
	let deviceStatesConnected: boolean = false;
	let unsubscribeDeviceInfos: Unsubscriber = null;
	let unsubscribeDeviceInfosConnected: Unsubscriber = null;
	let unsubscribeDeviceStates: Unsubscriber = null;
	let unsubscribeDeviceStatesConnected: Unsubscriber = null;

	onMount(async () => {
		console.log("DevicesPage onMount");
		unsubscribeDeviceInfos = deviceInfosStream.subscribeData(value => { deviceInfos = value; });
		unsubscribeDeviceInfosConnected = deviceInfosStream.subscribeConnected(value => { deviceInfosConnected = value; });
		unsubscribeDeviceStates = deviceStatesStream.subscribeData(value => { deviceStates = value; });
		unsubscribeDeviceStatesConnected = deviceStatesStream.subscribeConnected(value => { deviceStatesConnected = value; });
	});

	onDestroy(() => {
		console.log("DevicesPage onDestroy");
		unsubscribeDeviceInfos();
		unsubscribeDeviceInfosConnected();
		unsubscribeDeviceStates();
		unsubscribeDeviceStatesConnected();
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

<h2>Infos</h2>
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

<h2>States</h2>
{#each deviceStates as state (state.getDeviceId())}
<div class="column is-full">
	<div class="box">
		{state.getBridgeId()} {state.getDeviceId()} {state.getValuesList().length}
		{#each state.getValuesList() as value (value.getName())}
			<DeviceValue value={value} />
		{/each}
	</div>
</div>
{:else}
<div class="column is-full">
	<div class="box">
		No devices.
	</div>
</div>
{/each}
