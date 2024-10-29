<script lang="ts">
	import { ServiceRoot } from '$lib/components/wh/service';
	import { Button } from "$lib/components/ui/button";
	import {
		Service,
		Service_ServiceType,
		Value,
		FloatAttribute,
		IntAttribute,

		FloatValue

	} from '$lib/api/v1/clients/client_service_pb';
	import { Gauge, Power, Thermometer, Minus, Plus } from 'lucide-svelte';
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

	let actionSetHeatingSetpoint = async (ev: MouseEvent, adjustment: number) => {
		ev.stopPropagation();
		if (attrHeatingSetpoint !== undefined) {
			let val = attrHeatingSetpoint.value + adjustment;
			if (val < attrHeatingSetpoint.min) val = attrHeatingSetpoint.min;
			if (val > attrHeatingSetpoint.max) val = attrHeatingSetpoint.max;
			action([
				new Value({
					id: 'heating_setpoint',
					float: new FloatValue({
						value: val
					})
				})
			]);
		}
	};
</script>

{#if service.typ === Service_ServiceType.CLIMATE}
	<ServiceRoot title={title} alias={service.alias} online={online}>
		<span slot="icon">
			<div class="p-2 rounded-full bg-secondary text-secondary-foreground">
				<Thermometer/>
			</div>
		</span>
		<span slot="details">
			<div class="flex flex-row gap-2 rounded-lg p-0">
				{#if attrLocalTemperature !== undefined}
				<div class="flex flex-row gap-0 items-center">
					<!-- <Thermometer class="size-5"/> -->
					<p>
						{attrLocalTemperature.value.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })}°C
					</p>
				</div>
				{/if}
				{#if attrHeatingSetpoint !== undefined}
				<div class="flex flex-row gap-0.5 items-center">
					<!-- <Gauge class="text-muted-foreground size-5"/> -->
					<p class="text-muted-foreground">
						{attrHeatingSetpoint.value.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })}°C
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
		</span>
		<span slot="dialog-desktop">
		{#if attrHeatingSetpoint !== undefined}
			<p class="text-center">Heating Setpoint</p>
			<div class="p-4 pb-0">
				<div class="flex items-center justify-center space-x-2">
				<Button
					variant="outline"
					size="icon"
					class="size-12 shrink-0 rounded-full"
					on:click={(ev) => actionSetHeatingSetpoint(ev, -0.5)}
					disabled={attrHeatingSetpoint.value <= attrHeatingSetpoint.min}
				>
					<Minus class="size-5" />
					<span class="sr-only">Decrease</span>
				</Button>
				<div class="flex-1 text-center">
					<div class="flex justify-center content-start">
						<div class="text-4xl font-bold tracking-tighter">
							{attrHeatingSetpoint.value.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })}
							<span class="text-2xl uppercase text-muted-foreground">°C</span>
						</div>
					</div>
				</div>
				<Button
					variant="outline"
					size="icon"
					class="size-12 shrink-0 rounded-full"
					on:click={(ev) => actionSetHeatingSetpoint(ev, 0.5)}
					disabled={attrHeatingSetpoint.value >= attrHeatingSetpoint.max}
				>
					<Plus class="size-5" />
					<span class="sr-only">Increase</span>
				</Button>
				</div>
			</div>
		{/if}
		</span>
		<span slot="dialog-mobile">
		{#if attrHeatingSetpoint !== undefined}
			<p class="text-center">Heating Setpoint</p>
			<div class="p-4 pb-0">
				<div class="flex items-center justify-center space-x-2">
					<Button
						variant="outline"
						size="icon"
						class="size-12 shrink-0 rounded-full"
						on:click={(ev) => actionSetHeatingSetpoint(ev, -0.5)}
						disabled={attrHeatingSetpoint.value <= attrHeatingSetpoint.min}
					>
						<Minus class="size-5" />
						<span class="sr-only">Decrease</span>
					</Button>
					<div class="flex-1 text-center">
						<div class="flex justify-center content-start">
							<div class="text-4xl font-bold tracking-tighter">
								{attrHeatingSetpoint.value.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })}
								<span class="text-2xl uppercase text-muted-foreground">°C</span>
							</div>
						</div>
					</div>
					<Button
						variant="outline"
						size="icon"
						class="size-12 shrink-0 rounded-full"
						on:click={(ev) => actionSetHeatingSetpoint(ev, 0.5)}
						disabled={attrHeatingSetpoint.value >= attrHeatingSetpoint.max}
					>
						<Plus class="size-5" />
						<span class="sr-only">Increase</span>
					</Button>
				</div>
				<div class="mt-3 h-[30px]"></div>
			</div>
		{/if}
		</span>
	</ServiceRoot>
{:else}
	<p>ERROR Service Type {Service_ServiceType[service.typ]} is not CLIMATE</p>
{/if}
