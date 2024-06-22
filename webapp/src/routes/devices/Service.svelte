<script lang="ts">
	import { Service, Service_ServiceType, Value } from '$lib/api/v1/clients/client_service_pb';

	import { ScrollArea } from '$lib/components/ui/scroll-area/index.js';
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

	export let title: string | undefined = undefined;
	export let online: boolean;
	export let service: Service;
	export let onAction: ((serviceID: string, vals: Value[]) => Promise<void>) | undefined;
	export let expandable: boolean = true;

	let action = async (vals: Value[]) => {
		if (onAction) {
			onAction(service.id, vals);
		}
	};
</script>

{#if !(service.typ === Service_ServiceType.INFO || service.typ === Service_ServiceType.ONLINE)}
	{#if service.typ === Service_ServiceType.RELAY}
		<!-- <ServiceRelay online={online} service={service} onAction={onAction} /> -->
		<ServiceRelay2 {title} {online} {service} {onAction} />
	{:else if service.typ === Service_ServiceType.INPUT}
		<!-- <ServiceInput online={online} service={service} onAction={onAction} /> -->
		<ServiceInput2 {title} {online} {service} />
	{:else if service.typ === Service_ServiceType.LIGHTBULB}
		<ServiceLightbulb {title} {online} {service} {onAction} />
	{:else if service.typ === Service_ServiceType.BATTERY}
		<ServiceBattery {title} {online} {service} />
	{:else if service.typ === Service_ServiceType.CLIMATE}
		<ServiceClimate {title} {online} {service} {onAction} />
	{:else if service.typ === Service_ServiceType.BUTTON}
		<ServiceButton {title} {online} {service} {expandable} />
	{:else if service.typ === Service_ServiceType.ENVIRONMENT}
		<ServiceEnvironment {title} {online} {service} />
	{:else if service.typ === Service_ServiceType.CONTACT}
		<ServiceContact {title} {online} {service} />
	{:else}
		<div class={online ? '' : 'bg-muted'}>
			<div class="pb-3">
				<ServiceHeader id={service.id} alias={service.alias} />
			</div>
			<div>
				<ScrollArea class="w-auto whitespace-nowrap" orientation="horizontal">
					<div class="flex w-auto space-x-4">
						{#each service.attrs as attr, i}
							<BoxedAttribute {online} {attr} onAction={action} />
						{/each}
					</div>
				</ScrollArea>
			</div>
		</div>
	{/if}
{/if}
