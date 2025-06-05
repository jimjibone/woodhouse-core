<script lang="ts">
	import { ActionResponse, Service, Service_ServiceType, Value } from '$lib/api/v1/clients/client_service_pb';

	import { ScrollArea } from '$lib/components/ui/scroll-area/index.js';
	import { SendActionRequest, SendFavoriteRequest } from '$lib/stores';
	import ServiceHeader from './ServiceHeader.svelte';
	import ServiceInput2 from './ServiceInput2.svelte';
	import ServiceRelay2 from './ServiceRelay2.svelte';
	import BoxedAttribute from './BoxedAttribute.svelte';
	import ServiceLightbulb from './ServiceLightbulb.svelte';
	import ServiceBattery from './ServiceBattery.svelte';
	import ServiceClimate from './ServiceClimate.svelte';
	import ServiceButton from './ServiceButton.svelte';
	import ServiceEnvironment from './ServiceEnvironment.svelte';
	import ServiceContact from './ServiceContact.svelte';
	import ServiceUpdate from './ServiceUpdate.svelte';
	import ServiceEnum from './ServiceEnum.svelte';
	import ServiceCamera from './ServiceCamera.svelte';

	export let deviceID: string;
	export let title: string | undefined = undefined;
	export let online: boolean;
	export let service: Service;
	export let expandable: boolean = true;

	let onActionNoHandler = async (serviceID: string, vals: Value[]) => {
		SendActionRequest(deviceID, serviceID, vals, (response: ActionResponse) => {
			console.error("unused action response values:", response.toJsonString());
		});
	};

	let onActionWithHandler = async (serviceID: string, vals: Value[], responseHandler: (response: ActionResponse) => void) => {
		SendActionRequest(deviceID, serviceID, vals, responseHandler);
	};

	let legacyAction = async (vals: Value[]) => {
		onActionNoHandler(service.id, vals);
	};

	let handleSetFavorite = async (serviceID: string, fave: boolean) => {
		SendFavoriteRequest(deviceID, serviceID, fave);
	};
</script>

{#if !(service.typ === Service_ServiceType.INFO || service.typ === Service_ServiceType.ONLINE)}
	{#if service.typ === Service_ServiceType.RELAY}
		<ServiceRelay2 {title} {online} {service} onAction={onActionNoHandler} />
	{:else if service.typ === Service_ServiceType.INPUT}
		<ServiceInput2 {title} {online} {service} />
	{:else if service.typ === Service_ServiceType.LIGHTBULB}
		<ServiceLightbulb {title} {online} {service} onAction={onActionWithHandler} onSetFavorite={handleSetFavorite} />
	{:else if service.typ === Service_ServiceType.BATTERY}
		<ServiceBattery {title} {online} {service} onSetFavorite={handleSetFavorite} />
	{:else if service.typ === Service_ServiceType.CLIMATE}
		<ServiceClimate {title} {online} {service} onAction={onActionNoHandler} onSetFavorite={handleSetFavorite} />
	{:else if service.typ === Service_ServiceType.BUTTON}
		<ServiceButton {title} {online} {service} {expandable} onSetFavorite={handleSetFavorite} />
	{:else if service.typ === Service_ServiceType.ENVIRONMENT}
		<ServiceEnvironment {title} {online} {service} />
	{:else if service.typ === Service_ServiceType.CONTACT}
		<ServiceContact {title} {online} {service} onSetFavorite={handleSetFavorite} />
	{:else if service.typ === Service_ServiceType.UPDATE}
		<ServiceUpdate {title} {online} {service} />
	{:else if service.typ === Service_ServiceType.ENUM}
		<ServiceEnum {title} {online} {service} onAction={onActionNoHandler} onSetFavorite={handleSetFavorite} />
	{:else if service.typ === Service_ServiceType.CAMERA}
		<ServiceCamera deviceID={deviceID} {title} {online} {service} onAction={onActionWithHandler} onSetFavorite={handleSetFavorite} />
	{:else if expandable}
		<div class={online ? '' : 'bg-muted'}>
			<div class="pb-3">
				<ServiceHeader id={service.id} alias={service.alias} />
			</div>
			<div>
				<ScrollArea class="w-auto whitespace-nowrap" orientation="horizontal">
					<div class="flex w-auto space-x-4">
						{#each service.attrs as attr, i}
							<BoxedAttribute {online} {attr} onAction={legacyAction} />
						{/each}
					</div>
				</ScrollArea>
			</div>
		</div>
	{/if}
{/if}
