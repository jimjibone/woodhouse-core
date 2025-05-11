<script lang="ts">
	import {
		Service,
		Service_ServiceType,
		IntAttribute,
		FloatAttribute
	} from '$lib/api/v1/clients/client_service_pb';
	import { ServiceRoot } from '$lib/components/wh/service';
	import { BatteryFull, BatteryLow, BatteryMedium, BatteryWarning } from 'lucide-svelte';
	import { cn } from '$lib/utils.js';

	export let title: string | undefined = undefined;
	export let online: boolean;
	export let service: Service;
	export let onSetFavorite: ((serviceID: string, fave: boolean) => Promise<void>) | undefined;

	$: alias = title ? title + (service.alias !== '' ? ': ' + service.alias : '') : service.alias;
	let attrLevel: IntAttribute | undefined;
	let attrVoltage: FloatAttribute | undefined;
	let level = 0n;
	let favorite: boolean = false;

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
		favorite = service.favorite;
	}

	let handleSetFavorite = async(fave: boolean) => {
		if (onSetFavorite) {
			await onSetFavorite(service.id, fave);
		}
	};
</script>

{#if service.typ === Service_ServiceType.BATTERY}
	<ServiceRoot deviceName={title} online={online} service={service} onSetFavorite={handleSetFavorite}>
		<span slot="icon">
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
		</span>
		<span slot="details">
			<div class="flex h-full flex-col justify-center gap-0">
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
		</span>
		<span slot="dialog-desktop">
			{#if attrLevel !== undefined}
				<p class="text-center">Level</p>
				<div class="p-4 pb-0">
					<div class="flex items-center justify-center space-x-2">
						<div class="flex-1 text-center">
							<div class="flex justify-center content-start">
								<div class="text-4xl font-bold tracking-tighter">
									{attrLevel.value.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })}
									<span class="text-2xl uppercase text-muted-foreground">%</span>
								</div>
							</div>
						</div>
					</div>
				</div>
			{/if}
			{#if attrVoltage !== undefined}
				<p class="text-center">Voltage</p>
				<div class="p-4 pb-0">
					<div class="flex items-center justify-center space-x-2">
						<div class="flex-1 text-center">
							<div class="flex justify-center content-start">
								<div class="text-4xl font-bold tracking-tighter">
									{attrVoltage.value.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })}
									<span class="text-2xl uppercase text-muted-foreground">V</span>
								</div>
							</div>
						</div>
					</div>
				</div>
			{/if}
		</span>
		<span slot="dialog-mobile">
			{#if attrLevel !== undefined}
				<p class="text-center">Level</p>
				<div class="p-4 pb-0">
					<div class="flex items-center justify-center space-x-2">
						<div class="flex-1 text-center">
							<div class="flex justify-center content-start">
								<div class="text-4xl font-bold tracking-tighter">
									{attrLevel.value.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })}
									<span class="text-2xl uppercase text-muted-foreground">%</span>
								</div>
							</div>
						</div>
					</div>
				</div>
			{/if}
			{#if attrVoltage !== undefined}
				<p class="text-center">Voltage</p>
				<div class="p-4 pb-0">
					<div class="flex items-center justify-center space-x-2">
						<div class="flex-1 text-center">
							<div class="flex justify-center content-start">
								<div class="text-4xl font-bold tracking-tighter">
									{attrVoltage.value.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })}
									<span class="text-2xl uppercase text-muted-foreground">V</span>
								</div>
							</div>
						</div>
					</div>
				</div>
			{/if}
		</span>
	</ServiceRoot>
{:else}
	<p>ERROR Service Type {Service_ServiceType[service.typ]} is not BATTERY</p>
{/if}
