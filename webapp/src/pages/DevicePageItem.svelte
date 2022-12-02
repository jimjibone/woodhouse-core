<script lang="ts">
    import { formatISO9075, fromUnixTime } from 'date-fns';
	import DeviceValue from '../components/DeviceValue.svelte';
	import { DeviceRequest, DeviceResponse, DeviceValue as DeviceValueType } from '../api/device_pb';
	import { DeviceInfoState, sendDeviceRequest } from '../stores/devices';

	export let device: DeviceInfoState = null;

	$: bridgeID = device.info ? device.info.getBridgeId() : (device.state ? device.state.getBridgeId() : "<no bridge id>");
	$: deviceID = device.info ? device.info.getDeviceId() : (device.state ? device.state.getDeviceId() : "<no device id>");
	$: url = device.info ? device.info.getUrl() : "";
	$: online = device.state ? (device.state.getOnline() ? "online" : "offline") : "<no online state>";
	$: lastSeen = device.state ? (device.state.hasLastSeen() ? formatISO9075(fromUnixTime(device.state.getLastSeen().getSeconds())) : "<no time>") : "<no time>";

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

<div class="card">
	<div class="card-content">
		<div class="content">
			<p class="title is-4">{device.info != null ? device.info.getName() : "<no device info>"}</p>
			<p class="subtitle is-6">bridge: <code>{bridgeID}</code>, device: <code>{deviceID}</code>, {online}
				{#if url !== ""}
				, url: <a href="{url}">{url}</a>
				{/if}
				, last seen: {lastSeen}
			</p>
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
	</div>
</div>
