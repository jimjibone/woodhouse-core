<script lang="ts">
    import { formatISO9075, fromUnixTime } from 'date-fns';
	import DeviceValue from '../components/DeviceValue.svelte';
	import { DeviceRequest, DeviceResponse, DeviceValue as DeviceValueType } from '../api/device_pb';
	import { DeviceInfoState, sendDeviceRequest, setDeviceHidden } from '../stores/devices';
    import { SetDeviceHiddenRequest, SetDeviceHiddenResponse } from '../api/reactor_service_pb';

	export let device: DeviceInfoState = null;
	export let showHidden: boolean = true;

	$: bridgeID = device.info ? device.info.getBridgeId() : (device.state ? device.state.getBridgeId() : "<no bridge id>");
	$: deviceID = device.info ? device.info.getDeviceId() : (device.state ? device.state.getDeviceId() : "<no device id>");
	$: url = device.info ? device.info.getUrl() : "";
	$: online = device.state ? (device.state.getOnline() ? "online" : "offline") : "<no online state>";
	$: lastSeen = device.state ? (device.state.hasLastSeen() ? formatISO9075(fromUnixTime(device.state.getLastSeen().getSeconds())) : "<no time>") : "<no time>";
	$: hidden = device.info ? device.info.getHidden() : false;

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

		<div class="level">
			<div class="level-left">
				<div class="level-item">
					<p class="title is-5">{device.info != null ? device.info.getName() : "<no device info>"}</p>
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
							<svg xmlns="http://www.w3.org/2000/svg" class="checkIcon" viewBox="0 0 512 512"><title>Open</title><path d="M384 224v184a40 40 0 01-40 40H104a40 40 0 01-40-40V168a40 40 0 0140-40h167.48M336 64h112v112M224 288L440 72" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"/></svg>
						</div>
					</a>
				</div>
				{/if}
			</div>
		</div>

		{#if device.state}
		<div class="content is-inline-flex is-inline-spacing is-flex-wrap-wrap">
			{#each device.state.getValuesList() as value (value.getName())}
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

	.checkIcon {
		width: 14px;
		height: 14px;
		display: block;
	}
</style>
