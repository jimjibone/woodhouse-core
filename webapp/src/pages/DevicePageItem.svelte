<script lang="ts">
	import DeviceValue from '../components/DeviceValue.svelte';
	import { DeviceRequest, DeviceResponse, DeviceValue as DeviceValueType } from '../api/device_pb';
	import { DeviceInfoState, sendDeviceRequest } from '../stores/devices';

	export let device: DeviceInfoState = null;

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
			<p class="title is-4">{device.info != null ? device.info.getName() : "<no name>"}</p>
			{#if device.info}
			<p class="subtitle is-6">bridge: <code>{device.info.getBridgeId()}</code>, device: <code>{device.info.getDeviceId()}</code>
				{#if device.info.getUrl() !== ""}
				, url: <a href="{device.info.getUrl()}">{device.info.getUrl()}</a>
				{/if}
			</p>
			{/if}
		</div>

		{#if device.state}
		<div class="content is-inline-flex is-inline-spacing">
			{#each device.state.getValuesList() as value (value.getName())}
				<DeviceValue value={value} writable writer={onRequest} />
			{/each}
		</div>
		{/if}
	</div>
</div>
