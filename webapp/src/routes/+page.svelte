<script lang="ts">
	import { onDestroy } from 'svelte';
	import { DeviceAction, DeviceStore, type DeviceStoreType } from '$lib/stores';
	import ServiceComponent from './devices/Service.svelte';
	import { getDeviceInfo } from '$lib/apitools';
	import * as Menubar from "$lib/components/ui/menubar";
	import { Asterisk, Lightbulb, Thermometer } from 'lucide-svelte';
	import { cn } from "$lib/utils.js";
	import { Service_ServiceType } from '$lib/api/v1/clients/client_service_pb';

	let store: DeviceStoreType;
	const unsubscribe = DeviceStore.subscribe((val: DeviceStoreType) => store = val);
	onDestroy(unsubscribe);

	let filterServiceTypes: Service_ServiceType[] = [];
	let filterAll = () => {
		filterServiceTypes = [];
	};
	let filterLightbulb = () => {
		filterServiceTypes = [ Service_ServiceType.LIGHTBULB ];
	};
	let filterClimate = () => {
		filterServiceTypes = [ Service_ServiceType.CLIMATE, Service_ServiceType.ENVIRONMENT ];
	};
	$: showServiceType = (allowAny: boolean, srv_typ: Service_ServiceType) : boolean => {
		if (allowAny && filterServiceTypes.length === 0) {
			return true;
		}
		for (let i = 0; i < filterServiceTypes.length; i++) {
			if (filterServiceTypes[i] === srv_typ) {
				return true;
			}
		}
		return false;
	};
</script>

<header class="bg-background sticky top-0 z-10 flex h-[57px] items-center gap-1 border-b px-4">
	<h1 class="text-xl font-semibold">Dashboard{store.connected ? "" : " - Disconnected (backoff=" + store.backoff + "ms)"}</h1>
</header>

<Menubar.Root class="fixed bottom-5 self-center shadow-lg rounded-full h-12">
	<Menubar.Menu>
		<Menubar.Item on:click={filterAll} class={cn("rounded-full px-1.5 py-1.5 hover:bg-muted cursor-pointer", filterServiceTypes.length === 0 && "bg-secondary")}>
			<Asterisk class="size-6"/>
		</Menubar.Item>
		<Menubar.Item on:click={filterLightbulb} class={cn("rounded-full px-1.5 py-1.5 hover:bg-muted cursor-pointer", showServiceType(false, Service_ServiceType.LIGHTBULB) && "bg-secondary")}>
			<Lightbulb class="size-6"/>
		</Menubar.Item>
		<Menubar.Item on:click={filterClimate} class={cn("rounded-full px-1.5 py-1.5 hover:bg-muted cursor-pointer", showServiceType(false, Service_ServiceType.CLIMATE) && "bg-secondary")}>
			<Thermometer class="size-6"/>
		</Menubar.Item>
	</Menubar.Menu>
</Menubar.Root>

<main class="grid gap-4 p-4 md:grid-cols-2 lg:grid-cols-3">
	{#each store.devices as dev, i (dev.id)}
		{#each dev.services as srv, i (srv.id)}
			{#if showServiceType(true, srv.typ)}
			{@const info = getDeviceInfo(dev)}
			<ServiceComponent deviceID={dev.id} title={info.name} online={info.online} service={srv} expandable={false} onAction={(serviceID, val) => {
				return DeviceAction(dev.id, serviceID, val);
			}}/>
			{/if}
		{/each}
	{:else}
		<div>
			<p>No devices!</p>
		</div>
	{/each}
</main>
