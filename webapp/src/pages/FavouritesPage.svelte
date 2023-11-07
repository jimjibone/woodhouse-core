<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type { Unsubscriber } from 'svelte/store';
	import { devicesStream } from '../stores/devices';
	import type { DeviceInfoState } from '../stores/devices';
	import DevicePageItem from './DevicePageItem.svelte';

	let devices: DeviceInfoState[] = [];
	let connected: boolean = false;
	let unsubscribeDevices: Unsubscriber;
	let unsubscribeConnected: Unsubscriber;

	// $: devices = derived([devicesStream], ([$devices]) => {
	// 	$devices = $devices.sort((a, b) => {
	// 		const aName = a.info ? a.info.getName() : a.fullId
	// 		const bName = b.info ? b.info.getName() : b.fullId
	// 		return aName > bName ? 1 : (bName > aName ? -1 : 0)
	// 	}).filter(device => device.info.getFavourite());
	// });

	onMount(async () => {
		// unsubscribeDevices = devicesStream.subscribeData(value => { devices = value; });
		unsubscribeDevices = devicesStream.subscribe(value => {
			devices = value.sort((a, b) => {
				const aName = a.info ? a.info.getName() : a.fullId
				const bName = b.info ? b.info.getName() : b.fullId
				return aName > bName ? 1 : (bName > aName ? -1 : 0)
			}).filter(device => device.info ? device.info.getFavourite() : false);
		});
		unsubscribeConnected = devicesStream.subscribeConnected(value => { connected = value; });
	});

	onDestroy(() => {
		unsubscribeDevices();
		unsubscribeConnected();
	});
</script>

<div class="container is-fluid">
	<div class="block"></div>

	{#each devices as device (device.fullId)}
	<DevicePageItem device={device} showHidden={true}/>
	{:else}
	<div class="column is-full">
		<div class="box">
			No favourites.
		</div>
	</div>
	{/each}
</div>
