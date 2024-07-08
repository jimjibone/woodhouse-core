<script lang="ts">
	import {
		Service,
		Service_ServiceType,
		FloatAttribute,
	} from '$lib/api/v1/clients/client_service_pb';
	import { Power, Thermometer, Droplet, Gauge } from 'lucide-svelte';
	import { cn } from '$lib/utils.js';

	export let title: string | undefined = undefined;
	export let online: boolean;
	export let service: Service;

	$: alias = title ? title + (service.alias !== '' ? ': ' + service.alias : '') : service.alias;
	let attrTemperature: FloatAttribute | undefined;
	let attrHumidity: FloatAttribute | undefined;
	let attrPressure: FloatAttribute | undefined;

	$: {
		for (const attr of service.attrs) {
			if (attr.id === 'temperature') {
				attrTemperature = attr.float;
			} else if (attr.id === 'humidity') {
				attrHumidity = attr.float;
			} else if (attr.id === 'pressure') {
				attrPressure = attr.float;
			}
		}
	}
</script>

{#if service.typ === Service_ServiceType.ENVIRONMENT}
	<div
		class={cn(
			'rounded-lg border bg-card p-2 text-card-foreground shadow-sm',
			!online && 'bg-muted'
		)}
	>
		<div class="flex flex-row gap-2">
			<div class="shrink">
				<div class="grid h-full place-content-center">
					<div class="p-2 rounded-full bg-secondary text-secondary-foreground">
						<Thermometer/>
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
						{#if attrTemperature !== undefined}
						<div class="flex flex-row gap-0 items-center">
							<!-- <Thermometer class="size-4"/> -->
							<p>
								{attrTemperature.value.toLocaleString(undefined, { maximumFractionDigits: 0 })}°C
							</p>
						</div>
						{/if}
						{#if attrHumidity !== undefined}
						<div class="flex flex-row gap-0 items-center">
							<!-- <Droplet class="text-muted-foreground size-4"/> -->
							<p class="text-muted-foreground">
								{attrHumidity.value.toLocaleString(undefined, { maximumFractionDigits: 0 })}%
							</p>
						</div>
						{/if}
						{#if attrPressure !== undefined}
						<div class="flex flex-row gap-0.5 items-center">
							<!-- <Gauge class="text-muted-foreground size-4"/> -->
							<p class="text-muted-foreground">
								{attrPressure.value.toLocaleString(undefined, { maximumFractionDigits: 0 })}hPa
							</p>
						</div>
						{/if}
					</div>
				</div>
			</div>
		</div>
	</div>
{:else}
	<p>ERROR Service Type {Service_ServiceType[service.typ]} is not ENVIRONMENT</p>
{/if}
