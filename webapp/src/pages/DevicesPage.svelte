<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type { Unsubscriber } from 'svelte/store';
    import Chip from '../components/Chip.svelte';
	import { devicesStream } from '../stores/devices';
	import type { DeviceInfoState } from '../stores/devices';
	import DevicePageItem from './DevicePageItem.svelte';

	let devices: DeviceInfoState[] = [];
	let connected: boolean = false;
	let unsubscribeDevices: Unsubscriber;
	let unsubscribeConnected: Unsubscriber;
	let showHidden: boolean = true;

	onMount(async () => {
		// unsubscribeDevices = devicesStream.subscribeData(value => { devices = value; });
		unsubscribeDevices = devicesStream.subscribe(value => {
			devices = value.sort((a, b) => {
				const aName = a.info ? a.info.getName() : a.fullId
				const bName = b.info ? b.info.getName() : b.fullId
				return aName > bName ? 1 : (bName > aName ? -1 : 0)
			})
		});
		unsubscribeConnected = devicesStream.subscribeConnected(value => { connected = value; });
	});

	onDestroy(() => {
		unsubscribeDevices();
		unsubscribeConnected();
	});
</script>

<div class="container is-fluid">
	<div class="block">
		<Chip checked={showHidden} on:click={() => showHidden = !showHidden}>Show Hidden</Chip>
	</div>

	{#each devices as device (device.fullId)}
	<DevicePageItem device={device} showHidden={showHidden}/>
	{:else}
	<div class="column is-full">
		<div class="box">
			No devices.
		</div>
	</div>
	{/each}
</div>
