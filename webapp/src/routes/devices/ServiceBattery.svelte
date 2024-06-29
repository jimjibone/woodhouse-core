<script lang="ts">
	import {
		Service,
		Service_ServiceType,
		Value,
		IntValue,
		Attribute as AttributeType,
		IntAttribute,
		FloatAttribute
	} from '$lib/api/v1/clients/client_service_pb';
	import { Battery, BatteryFull, BatteryLow, BatteryMedium, BatteryWarning } from 'lucide-svelte';
	import { cn } from '$lib/utils.js';
	import { validators } from 'tailwind-merge';

	export let title: string | undefined = undefined;
	export let online: boolean;
	export let service: Service;

	$: alias = title ? title + (service.alias !== '' ? ': ' + service.alias : '') : service.alias;
	let attrLevel: IntAttribute | undefined;
	let attrVoltage: FloatAttribute | undefined;
	let level = 0n;

	$: {
		for (const attr of service.attrs) {
			if (attr.id === 'level') {
				attrLevel = attr.int;
				if (attrLevel?.value) {
					level = attrLevel.value;
				}
			} else if (attr.id === 'voltage') {
				attrVoltage = attr.float;
			}
		}
	}
</script>

{#if service.typ === Service_ServiceType.BATTERY}
	<!-- <div class="grid grid-cols-2 gap-4"> -->
	<div
		class={cn(
			'rounded-lg border bg-card p-2 text-card-foreground shadow-sm',
			!online && 'bg-muted'
		)}
	>
		<div class="flex flex-row gap-2">
			<div class="shrink">
				<div class="grid h-full place-content-center">
					<div class={cn("p-2 rounded-full", level < 10 ? "bg-red-400 text-black" : "bg-secondary text-secondary-foreground")}>
					{#if level < 20}
					<BatteryWarning />
					{:else if level < 33}
					<BatteryLow/>
					{:else if level < 66}
					<BatteryMedium/>
					{:else}
					<BatteryFull/>
					{/if}
					</div>
				</div>
			</div>
			<div class="grow">
				<div class="flex h-full flex-col justify-center gap-0">
					{#if alias !== ''}
						<div class="rounded-lg p-0">
							<p class="font-semibold">{alias}</p>
						</div>
					{/if}
					<div class="flex flex-row gap-2 rounded-lg p-0">
						{#if attrLevel !== undefined}
							<p>
								{attrLevel.value.toLocaleString(undefined, { maximumFractionDigits: 0 })}%
							</p>
						{/if}
						{#if attrVoltage !== undefined}
							<p class="text-muted-foreground">
								{attrVoltage.value.toLocaleString(undefined, { maximumFractionDigits: 1 })}V
							</p>
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
	<p>ERROR Service Type {Service_ServiceType[service.typ]} is not BATTERY</p>
{/if}
