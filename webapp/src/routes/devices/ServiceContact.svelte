<script lang="ts">
	import { Service, Service_ServiceType, BoolAttribute } from '$lib/api/v1/clients/client_service_pb';
	import { ServiceRoot } from '$lib/components/wh/service';
	import { DoorClosed, DoorOpen } from 'lucide-svelte';
	import { cn } from "$lib/utils.js";

	export let title: string | undefined = undefined;
	export let online: boolean;
	export let service: Service;
	export let onSetFavorite: ((serviceID: string, fave: boolean) => Promise<void>) | undefined;

	$:alias = (title ? title + (service.alias !== "" ? ": "+service.alias : "") : service.alias);
	let attrClosed: BoolAttribute | undefined

	let displayOn: boolean = false;

	$:{
		for (const attr of service.attrs) {
			if (attr.id === "closed") {
				attrClosed = attr.bool;
			}
		}
		if (online && !attrClosed?.value) {
			displayOn = true;
		} else {
			displayOn = false;
		}
	}
</script>

{#if service.typ === Service_ServiceType.CONTACT}
	<ServiceRoot deviceName={title} online={online} service={service} {onSetFavorite}>
		<span slot="icon">
			<div class={cn("p-2 rounded-full", displayOn ? "bg-blue-400 dark:bg-blue-600 text-secondary-foreground" : "bg-secondary text-secondary-foreground")}>
				{#if attrClosed?.value}
					<DoorClosed/>
				{:else}
					<DoorOpen/>
				{/if}
			</div>
		</span>
		<span slot="details">
			{#if attrClosed !== undefined}
				<p class="">{attrClosed.value ? "Closed" : "Open"}</p>
			{/if}
		</span>
	</ServiceRoot>
{:else}
<p>ERROR Service Type {Service_ServiceType[service.typ]} is not CONTACT</p>
{/if}
