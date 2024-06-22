<script lang="ts">
	import { Service, Service_ServiceType, BoolAttribute } from '$lib/api/v1/clients/client_service_pb';
	import { DoorClosed, DoorOpen } from 'lucide-svelte';
	import { cn } from "$lib/utils.js";

	export let title: string | undefined = undefined;
	export let online: boolean;
	export let service: Service;

	$:alias = (title ? title + (service.alias !== "" ? ": "+service.alias : "") : service.alias);
	let attrClosed: BoolAttribute | undefined

	$:{
		for (const attr of service.attrs) {
			if (attr.id === "closed") {
				attrClosed = attr.bool;
			}
		}
	}
</script>

{#if service.typ === Service_ServiceType.CONTACT}
<!-- <div class="grid grid-cols-2 gap-4"> -->
<div class={cn("p-2 rounded-lg border bg-card text-card-foreground shadow-sm", !online && "bg-muted")}>
	<div class="flex flex-row gap-2">
		<div class="shrink">
			<div class="h-full grid place-content-center">
				{#if attrClosed?.value}
				<div class="p-2 rounded-full bg-secondary text-secondary-foreground">
					<DoorClosed/>
				</div>
				{:else}
				<div class="p-2 rounded-full bg-yellow-400 text-black">
					<DoorOpen/>
				</div>
				{/if}
			</div>
		</div>
		<div class="grow">
			<div class="h-full flex flex-col gap-0 justify-center">
				{#if alias !== ""}
				<div class="p-0 rounded-lg">
					<p class="font-semibold">{alias}</p>
				</div>
				{/if}
				<div class="p-0 rounded-lg flex flex-row gap-2">
					{#if attrClosed !== undefined}
					<p class="">{attrClosed.value ? "Closed" : "Open"}</p>
					{/if}
				</div>
			</div>
		</div>
	</div>
</div>
<!-- <div class="p-4 rounded-lg shadow-lg bg-fuchsia-500">02</div>
<div class="p-4 rounded-lg shadow-lg bg-fuchsia-500">03</div>
</div> -->
{:else}
<p>ERROR Service Type {Service_ServiceType[service.typ]} is not CONTACT</p>
{/if}
