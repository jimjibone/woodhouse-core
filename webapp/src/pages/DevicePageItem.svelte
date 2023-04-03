<script lang="ts">
    import { formatISO9075, fromUnixTime } from 'date-fns';
	import DeviceValue from '../components/DeviceValue.svelte';
	import { DeviceRequest, DeviceResponse, DeviceValue as DeviceValueType } from '../api/device_pb';
	import { DeviceInfoState, sendDeviceRequest, setDeviceFavourite, setDeviceHidden } from '../stores/devices';
    import { SetDeviceFavouriteRequest, SetDeviceFavouriteResponse, SetDeviceHiddenRequest, SetDeviceHiddenResponse } from '../api/reactor_service_pb';

	export let device: DeviceInfoState = null;
	export let showHidden: boolean = true;
	$: values = device.state ? device.state.getValuesList().sort((a, b) => {
		const aName = a.getName()
		const bName = b.getName()
		return aName > bName ? 1 : (bName > aName ? -1 : 0)
	}) : []

	$: bridgeID = device.info ? device.info.getBridgeId() : (device.state ? device.state.getBridgeId() : "<no bridge id>");
	$: deviceID = device.info ? device.info.getDeviceId() : (device.state ? device.state.getDeviceId() : "<no device id>");
	$: url = device.info ? device.info.getUrl() : "";
	$: online = device.state ? device.state.getOnline() : false;
	$: lastSeen = device.state ? (device.state.hasLastSeen() ? formatISO9075(fromUnixTime(device.state.getLastSeen().getSeconds())) : "<no time>") : "<no time>";
	$: hidden = device.info ? device.info.getHidden() : false;
	$: favourite = device.info ? device.info.getFavourite() : false;

	function toggleHidden() : void {
		const req = new SetDeviceHiddenRequest();
		req.setBridgeId(device.info.getBridgeId());
		req.setDeviceId(device.info.getDeviceId());
		req.setHidden(!hidden);
		console.log(`${device.fullId} request:`, req.toObject());
		setDeviceHidden(req)
		.then((res: SetDeviceHiddenResponse) => {
			console.log(`${device.fullId} response:`, res.toObject());
		}).catch((err: any) => {
			console.error(`${device.fullId} response:`, err);
		});
	}

	function toggleFavourite() : void {
		const req = new SetDeviceFavouriteRequest();
		req.setBridgeId(device.info.getBridgeId());
		req.setDeviceId(device.info.getDeviceId());
		req.setFavourite(!favourite);
		console.log(`${device.fullId} request:`, req.toObject());
		setDeviceFavourite(req)
		.then((res: SetDeviceFavouriteResponse) => {
			console.log(`${device.fullId} response:`, res.toObject());
		}).catch((err: any) => {
			console.error(`${device.fullId} response:`, err);
		});
	}

	function onRequest(v: DeviceValueType) : void {
		const req = new DeviceRequest();
		req.setBridgeId(device.info.getBridgeId());
		req.setDeviceId(device.info.getDeviceId());
		req.setValuesList([v]);
		console.log(`${device.fullId} request:`, req.toObject());
		sendDeviceRequest(req)
		.then((res: DeviceResponse) => {
			console.log(`${device.fullId} response:`, res.toObject());
		}).catch((err: any) => {
			console.error(`${device.fullId} response:`, err);
		});
	}
</script>

{#if !hidden || showHidden}
<div class="block">
	<!-- <div class="card-content"> -->
		<!-- <div class="content">
			<p class="title is-6">{device.info != null ? device.info.getName() : "<no device info>"}</p>
			<p class="subtitle is-6">bridge: <code>{bridgeID}</code>, device: <code>{deviceID}</code>, {online}
				{#if url !== ""}
				, url: <a href="{url}">{url}</a>
				{/if}
				, last seen: {lastSeen}
				{#if hidden}
				<button on:click={toggleHidden}>, hidden</button>
				{:else}
				<button on:click={toggleHidden}>, visible</button>
				{/if}
			</p>
		</div> -->

		<!-- title: desktop -->
		<div class="level is-hidden-mobile">
			<div class="level-left">
				<div class="level-item">
					<p class="title is-5">{device.info != null ? device.info.getName() : "<no device info>"}</p>
				</div>
				{#if !online}
				<div class="level-item">
					<p class="tag is-danger">offline</p>
				</div>
				{/if}
				{#if showHidden}
				<div class="level-item">
					<button class="button tag is-light" on:click={toggleHidden}>
						<div class="iconWrapper">
							<!-- https://ionic.io/ionicons -->
							{#if hidden}
							<svg xmlns="http://www.w3.org/2000/svg" class="ionicon" viewBox="0 0 512 512"><title>Eye Off</title><path d="M432 448a15.92 15.92 0 01-11.31-4.69l-352-352a16 16 0 0122.62-22.62l352 352A16 16 0 01432 448zM255.66 384c-41.49 0-81.5-12.28-118.92-36.5-34.07-22-64.74-53.51-88.7-91v-.08c19.94-28.57 41.78-52.73 65.24-72.21a2 2 0 00.14-2.94L93.5 161.38a2 2 0 00-2.71-.12c-24.92 21-48.05 46.76-69.08 76.92a31.92 31.92 0 00-.64 35.54c26.41 41.33 60.4 76.14 98.28 100.65C162 402 207.9 416 255.66 416a239.13 239.13 0 0075.8-12.58 2 2 0 00.77-3.31l-21.58-21.58a4 4 0 00-3.83-1 204.8 204.8 0 01-51.16 6.47zM490.84 238.6c-26.46-40.92-60.79-75.68-99.27-100.53C349 110.55 302 96 255.66 96a227.34 227.34 0 00-74.89 12.83 2 2 0 00-.75 3.31l21.55 21.55a4 4 0 003.88 1 192.82 192.82 0 0150.21-6.69c40.69 0 80.58 12.43 118.55 37 34.71 22.4 65.74 53.88 89.76 91a.13.13 0 010 .16 310.72 310.72 0 01-64.12 72.73 2 2 0 00-.15 2.95l19.9 19.89a2 2 0 002.7.13 343.49 343.49 0 0068.64-78.48 32.2 32.2 0 00-.1-34.78z"/><path d="M256 160a95.88 95.88 0 00-21.37 2.4 2 2 0 00-1 3.38l112.59 112.56a2 2 0 003.38-1A96 96 0 00256 160zM165.78 233.66a2 2 0 00-3.38 1 96 96 0 00115 115 2 2 0 001-3.38z"/></svg>
							{:else}
							<svg xmlns="http://www.w3.org/2000/svg" class="ionicon" viewBox="0 0 512 512"><title>Eye</title><path d="M255.66 112c-77.94 0-157.89 45.11-220.83 135.33a16 16 0 00-.27 17.77C82.92 340.8 161.8 400 255.66 400c92.84 0 173.34-59.38 221.79-135.25a16.14 16.14 0 000-17.47C428.89 172.28 347.8 112 255.66 112z" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"/><circle cx="256" cy="256" r="80" fill="none" stroke="currentColor" stroke-miterlimit="10" stroke-width="32"/></svg>
							{/if}
						</div>
					</button>
				</div>
				{/if}
				<div class="level-item">
					<button class="button tag is-light" on:click={toggleFavourite}>
						<div class="iconWrapper">
							<!-- https://ionic.io/ionicons -->
							{#if favourite}
							<svg xmlns="http://www.w3.org/2000/svg" class="ionicon" viewBox="0 0 512 512"><title>Star</title><path d="M394 480a16 16 0 01-9.39-3L256 383.76 127.39 477a16 16 0 01-24.55-18.08L153 310.35 23 221.2a16 16 0 019-29.2h160.38l48.4-148.95a16 16 0 0130.44 0l48.4 149H480a16 16 0 019.05 29.2L359 310.35l50.13 148.53A16 16 0 01394 480z"/></svg>
							{:else}
							<svg xmlns="http://www.w3.org/2000/svg" class="ionicon" viewBox="0 0 512 512"><title>Star</title><path d="M480 208H308L256 48l-52 160H32l140 96-54 160 138-100 138 100-54-160z" fill="none" stroke="currentColor" stroke-linejoin="round" stroke-width="32"/></svg>
							{/if}
						</div>
					</button>
				</div>
			</div>
			<div class="level-right">
				<div class="level-item">
					<div class="field is-grouped">
						<div class="control">
							<div class="tags has-addons">
								<span class="tag is-white">{bridgeID}</span>
								<span class="tag is-light">{deviceID}</span>
							</div>
						</div>
					</div>
				</div>
				<div class="level-item">
					<span class="tag is-light">{lastSeen}</span>
				</div>
				{#if url !== ""}
				<div class="level-item">
					<a class="button tag is-light" href="{url}" target="_blank" rel="noopener noreferrer">
						<div class="iconWrapper">
							<!-- https://ionic.io/ionicons -->
							<svg xmlns="http://www.w3.org/2000/svg" class="ionicon" viewBox="0 0 512 512"><title>Open</title><path d="M384 224v184a40 40 0 01-40 40H104a40 40 0 01-40-40V168a40 40 0 0140-40h167.48M336 64h112v112M224 288L440 72" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"/></svg>
						</div>
					</a>
				</div>
				{/if}
			</div>
		</div>
		<!-- title: mobile -->
		<div class="level is-hidden-tablet is-mobile">
			<div class="level-left">
				<div class="level-item">
					<p class="title is-5">{device.info != null ? device.info.getName() : "<no device info>"}</p>
				</div>
			</div>
			<div class="level-right">
				{#if !online}
				<div class="level-item">
					<p class="tag is-danger">offline</p>
				</div>
				{/if}
			</div>
		</div>

		{#if device.state}
		<div class="content is-inline-flex is-inline-spacing is-flex-wrap-wrap">
			{#each values as value (value.getName())}
				<DeviceValue value={value} writable writer={onRequest} />
			{:else}
			<p>No values!</p>
			{/each}
		</div>
		{:else}
		<div class="content">
			<p>No state!</p>
		</div>
		{/if}
	<!-- </div> -->
</div>
{/if}

<style>
	.iconWrapper {
		width: 14px;
		max-width: 14px;
		height: 14px;
		display: inline-block;
		vertical-align: middle;
		/* overflow-x: hidden;
		overflow-y: hidden; */
		/* color: #4a4a4a; */
	}

	.ionicon {
		width: 14px;
		height: 14px;
		display: block;
	}
</style>
