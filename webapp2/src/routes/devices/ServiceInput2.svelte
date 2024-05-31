<script lang="ts">
	import { Service, Service_ServiceType, Value, BoolValue, Attribute as AttributeType, BoolAttribute } from '$lib/api/v1/clients/client_service_pb';
	import { LogIn } from 'lucide-svelte';
	import { cn } from "$lib/utils.js";
	import { validators } from 'tailwind-merge';

	export let title: string | undefined = undefined;
	export let online: boolean;
	export let service: Service;

	let alias: string = (title ? title + (service.alias !== "" ? ": "+service.alias : "") : service.alias);
	let attrOn: BoolAttribute | undefined
	let attrOthers: AttributeType[]

	$:{
		attrOthers = [];
		for (const attr of service.attrs) {
			if (attr.id === "on") {
				attrOn = attr.bool;
			} else {
				attrOthers = [...attrOthers, attr];
			}
		}
	}
</script>

{#if service.typ === Service_ServiceType.INPUT}
<!-- <div class="grid grid-cols-2 gap-4"> -->
<div class={cn("p-2 rounded-lg border bg-card text-card-foreground shadow-sm", !online && "bg-muted")}>
	<div class="flex flex-row gap-2">
		<div class="shrink">
			<div class="h-full grid place-content-center">
				<div class={cn("p-2 rounded-full", attrOn?.value ? "bg-yellow-400 text-black" : "bg-secondary text-secondary-foreground")}>
					<LogIn/>
				</div>
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
					{#if attrOn !== undefined}
					<p class="">{attrOn.value ? "On" : "Off"}</p>
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
<p>ERROR Service Type {Service_ServiceType[service.typ]} is not INPUT</p>
{/if}
