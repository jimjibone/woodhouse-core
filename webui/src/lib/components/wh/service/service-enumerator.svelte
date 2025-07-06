<script lang="ts">
	import { onDestroy } from 'svelte';
	import { DevicesStore, type DevicesStoreType } from '$lib/stores/devices-stream';
	import { ServiceRoot, BatteryService, ClimateService, EnumService, LightbulbService } from '$lib/components/wh/service';
	import { Service_ServiceType, type Service } from '$lib/api/v1/clients/client_service_pb';

	let {
		deviceName,
		showDeviceName,
		deviceID,
		online,
		service,
		class: className
	}: {
		deviceName: string,
		showDeviceName?: boolean,
		deviceID: string,
		online: boolean,
		service: Service,
		class?: string | undefined
	} = $props();
</script>

{#if
	service.typ !== Service_ServiceType.INFO &&
	service.typ !== Service_ServiceType.ONLINE
}
	<div class={className}>
		{#if service.typ == Service_ServiceType.BATTERY}
			<BatteryService
				{deviceName}
				{showDeviceName}
				{deviceID}
				{online}
				{service}/>
		{:else if service.typ == Service_ServiceType.CLIMATE}
			<ClimateService
				{deviceName}
				{showDeviceName}
				{deviceID}
				{online}
				{service}/>
		{:else if service.typ == Service_ServiceType.ENUM}
			<EnumService
				{deviceName}
				{showDeviceName}
				{deviceID}
				{online}
				{service}/>
		{:else if service.typ == Service_ServiceType.LIGHTBULB}
			<LightbulbService
				{deviceName}
				{showDeviceName}
				{deviceID}
				{online}
				{service}/>
		{:else}
			<ServiceRoot
				{deviceName}
				{showDeviceName}
				{deviceID}
				{online}
				{service}
				actionPending={false}
				errorSignal={null}>
			</ServiceRoot>
		{/if}
	</div>
{/if}
