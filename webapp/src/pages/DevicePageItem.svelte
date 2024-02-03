<script lang="ts">
    import { formatISO9075, fromUnixTime } from 'date-fns';
	import DeviceValue from '../components/DeviceValue.svelte';
	import { DeviceRequest, DeviceResponse, DeviceValue as DeviceValueType } from '../api/device_pb';
	import { sendDeviceRequest, setDeviceFavourite, setDeviceHidden } from '../stores/devices';
	import type { DeviceInfoState } from '../stores/devices';
    import { SetDeviceFavouriteRequest, SetDeviceFavouriteResponse, SetDeviceHiddenRequest, SetDeviceHiddenResponse } from '../api/reactor_service_pb';

	import { Badge } from "$lib/components/ui/badge";
	import { Button } from "$lib/components/ui/button";
	import * as DropdownMenu from "$lib/components/ui/dropdown-menu";
	import { ExternalLink, Eye, EyeOff, Star, StarOff, MoreVertical } from 'lucide-svelte';

	export let device: DeviceInfoState;
	export let showHidden: boolean = true;
	$: values = device.state ? device.state.getValuesList().sort((a, b) => {
		const aName = a.getName()
		const bName = b.getName()
		return aName > bName ? 1 : (bName > aName ? -1 : 0)
	}) : []

	$: bridgeID = device.info ? device.info.getBridgeId() : (device.state ? device.state.getBridgeId() : "<no bridge id>");
	$: deviceID = device.info ? device.info.getDeviceId() : (device.state ? device.state.getDeviceId() : "<no device id>");
	$: deviceName = device.info ? (device.info.getName() !== "" ? device.info.getName() : "<no device name>") : (device.state ? device.state.getDeviceId() : "<no device info>");
	$: hasDeviceName = device.info ? device.info.getName() !== "" : false;
	$: url = device.info ? device.info.getUrl() : "";
	$: online = device.state ? device.state.getOnline() : false;
	// @ts-ignore: device.state.getLastSeen() may be undefined
	$: lastSeen = device.state ? (device.state.hasLastSeen() ? formatISO9075(fromUnixTime(device.state.getLastSeen().getSeconds())) : "<no time>") : "<no time>";
	$: hidden = device.info ? device.info.getHidden() : false;
	$: favourite = device.info ? device.info.getFavourite() : false;

	function toggleHidden() : void {
		const req = new SetDeviceHiddenRequest();
		req.setBridgeId(bridgeID);
		req.setDeviceId(deviceID);
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
		req.setBridgeId(bridgeID);
		req.setDeviceId(deviceID);
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
		req.setBridgeId(bridgeID);
		req.setDeviceId(deviceID);
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
					<h3 class="text-xl font-semibold tracking-tight">{deviceName}</h3>
				</div>
				{#if online}
				<Badge variant="outline">online</Badge>
				{:else}
				<Badge variant="destructive">offline</Badge>
				{/if}
			</div>
			<div class="level-right">
				<div class="level-item">
					<Badge variant="outline">{bridgeID}</Badge>
					<Badge variant="outline">{deviceID}</Badge>
					<Badge variant="outline">{lastSeen}</Badge>
				</div>
				<DropdownMenu.Root preventScroll={false}>
					<DropdownMenu.Trigger asChild let:builder>
						<Button builders={[builder]} variant="ghost" size="sm">
							<MoreVertical size={18} />
						</Button>
					</DropdownMenu.Trigger>
					<DropdownMenu.Content class="w-56">
						<DropdownMenu.Item on:click={toggleHidden}>
							{#if hidden}
							<span>Un-hide</span>
							{:else}
							<span>Hide</span>
							{/if}
						</DropdownMenu.Item>
						<DropdownMenu.Item on:click={toggleFavourite}>
							{#if favourite}
							<span>Un-favourite</span>
							{:else}
							<span>Favourite</span>
							{/if}
						</DropdownMenu.Item>
						{#if url !== ""}
						<DropdownMenu.Item href="{url}" target="_blank" rel="noopener noreferrer">
							<span>Open Page</span>
						</DropdownMenu.Item>
						{/if}
					</DropdownMenu.Content>
				</DropdownMenu.Root>
			</div>
		</div>
		<!-- title: mobile -->
		<h3 class="text-xl font-semibold tracking-tight pb-3 is-hidden-tablet">{deviceName}</h3>
		<div class="level is-hidden-tablet is-mobile">
			<div class="level-left">
				<div class="level-item">
					{#if online}
					<Badge variant="outline">online</Badge>
					{:else}
					<Badge variant="destructive">offline</Badge>
					{/if}
				</div>
				<div class="level-item">
					<Badge variant="outline">{lastSeen}</Badge>
				</div>
				{#if url !== ""}
				<div class="level-item">
					<DropdownMenu.Root preventScroll={false}>
						<DropdownMenu.Trigger asChild let:builder>
							<Button builders={[builder]} variant="ghost" size="sm">
								<MoreVertical size={18} />
							</Button>
						</DropdownMenu.Trigger>
						<DropdownMenu.Content class="w-56">
							<DropdownMenu.Item on:click={toggleHidden}>
								{#if hidden}
								<span>Un-hide</span>
								{:else}
								<span>Hide</span>
								{/if}
							</DropdownMenu.Item>
							<DropdownMenu.Item on:click={toggleFavourite}>
								{#if favourite}
								<span>Un-favourite</span>
								{:else}
								<span>Favourite</span>
								{/if}
							</DropdownMenu.Item>
							{#if url !== ""}
							<DropdownMenu.Item href="{url}" target="_blank" rel="noopener noreferrer">
								<span>Open Page</span>
							</DropdownMenu.Item>
							{/if}
						</DropdownMenu.Content>
					</DropdownMenu.Root>
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
