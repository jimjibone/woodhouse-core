<script lang="ts">
	import {
		Service,
		Service_ServiceType,
		Value,
		FloatAttribute,
		IntAttribute
	} from '$lib/api/v1/clients/client_service_pb';
	import { Gauge, Power, Thermometer } from 'lucide-svelte';
	import { cn } from '$lib/utils.js';

	export let title: string | undefined = undefined;
	export let online: boolean;
	export let service: Service;
	export let onAction: ((serviceID: string, vals: Value[]) => Promise<void>) | undefined;

	$: alias = title ? title + (service.alias !== '' ? ': ' + service.alias : '') : service.alias;
	let attrHeatingSetpoint: FloatAttribute | undefined;
	let attrLocalTemperature: FloatAttribute | undefined;
	let attrPIHeatingDemand: IntAttribute | undefined;
	let actionPending: boolean = false;

	$: {
		for (const attr of service.attrs) {
			if (attr.id === 'heating_setpoint') {
				attrHeatingSetpoint = attr.float;
			} else if (attr.id === 'local_temperature') {
				attrLocalTemperature = attr.float;
			} else if (attr.id === 'pi_heating_demand') {
				attrPIHeatingDemand = attr.int;
			}
		}
	}

	let action = async (vals: Value[]) => {
		if (onAction) {
			actionPending = true;
			await onAction(service.id, vals);
			actionPending = false;
		}
	};

	// let actionOn = async (val: boolean) => {
	// 	action([
	// 		new Value({
	// 			id: 'on',
	// 			bool: new BoolValue({
	// 				value: val
	// 			})
	// 		})
	// 	]);
	// };
</script>

{#if service.typ === Service_ServiceType.CLIMATE}
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
						{#if attrLocalTemperature !== undefined}
						<div class="flex flex-row gap-0 items-center">
							<!-- <Thermometer class="size-5"/> -->
							<p>
								{attrLocalTemperature.value.toLocaleString(undefined, { maximumFractionDigits: 0 })}°C
							</p>
						</div>
						{/if}
						{#if attrHeatingSetpoint !== undefined}
						<div class="flex flex-row gap-0.5 items-center">
							<!-- <Gauge class="text-muted-foreground size-5"/> -->
							<p class="text-muted-foreground">
								{attrHeatingSetpoint.value.toLocaleString(undefined, { maximumFractionDigits: 0 })}°C
							</p>
						</div>
						{/if}
						{#if attrPIHeatingDemand !== undefined}
						<div class="flex flex-row gap-0.5 items-center">
							<!-- <Power class="text-muted-foreground size-5"/> -->
							<p class="text-muted-foreground">
								{attrPIHeatingDemand.value.toLocaleString(undefined, { maximumFractionDigits: 0 })}%
							</p>
						</div>
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
	<p>ERROR Service Type {Service_ServiceType[service.typ]} is not CLIMATE</p>
{/if}
