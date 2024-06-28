<script lang="ts">
	import {
		Service,
		Service_ServiceType,
		Value,
		BoolValue,
		Attribute as AttributeType,
		BoolAttribute,
		FloatAttribute
	} from '$lib/api/v1/clients/client_service_pb';
	import { Loader, Power, PowerOff } from 'lucide-svelte';
	import { cn } from '$lib/utils.js';

	export let title: string | undefined = undefined;
	export let online: boolean;
	export let service: Service;
	export let onAction: ((serviceID: string, vals: Value[]) => Promise<void>) | undefined;

	$: alias = title ? title + (service.alias !== '' ? ': ' + service.alias : '') : service.alias;
	let attrOn: BoolAttribute | undefined;
	let attrVoltage: FloatAttribute | undefined;
	let attrCurrent: FloatAttribute | undefined;
	let attrPower: FloatAttribute | undefined;
	let attrTemperature: FloatAttribute | undefined;
	let attrOthers: AttributeType[];
	let actionPending: boolean = false;

	$: {
		attrOthers = [];
		for (const attr of service.attrs) {
			if (attr.id === 'on') {
				attrOn = attr.bool;
			} else if (attr.id === 'voltage') {
				attrVoltage = attr.float;
			} else if (attr.id === 'current') {
				attrCurrent = attr.float;
			} else if (attr.id === 'power') {
				attrPower = attr.float;
			} else if (attr.id === 'temperature') {
				attrTemperature = attr.float;
			} else {
				attrOthers = [...attrOthers, attr];
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

	let actionOn = async (val: boolean) => {
		action([
			new Value({
				id: 'on',
				bool: new BoolValue({
					value: val
				})
			})
		]);
	};

	let actionOnToggle = async () => {
		if (attrOn !== undefined) {
			actionOn(!attrOn.value);
		}
	};
</script>

{#if service.typ === Service_ServiceType.RELAY}
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
					<button
						class={cn(
							'rounded-full p-2',
							attrOn?.value ? 'bg-green-400 dark:bg-green-600 text-secondary-foreground' : 'bg-secondary text-secondary-foreground'
						)}
						on:click={actionOnToggle}
						disabled={actionPending}
					>
						{#if actionPending}
							<Loader />
						{:else if attrOn?.value}
							<Power />
						{:else}
							<PowerOff />
						{/if}
					</button>
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
						{#if attrOn !== undefined}
							<p>{attrOn.value ? 'On' : 'Off'}</p>
						{/if}
						{#if attrVoltage !== undefined}
							<p class="text-muted-foreground">
								{attrVoltage.value.toLocaleString(undefined, { maximumFractionDigits: 0 })}V
							</p>
						{/if}
						{#if attrPower !== undefined}
							<p class="text-muted-foreground">
								{attrPower.value.toLocaleString(undefined, { maximumFractionDigits: 0 })}W
							</p>
						{/if}
						{#if attrTemperature !== undefined}
							<p class="text-muted-foreground">
								{attrTemperature.value.toLocaleString(undefined, { maximumFractionDigits: 0 })}°C
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
	<p>ERROR Service Type {Service_ServiceType[service.typ]} is not RELAY</p>
{/if}
