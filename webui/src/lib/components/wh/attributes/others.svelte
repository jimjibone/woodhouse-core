<script lang="ts">
	import {
		BoolValueSchema,
		ValueSchema,
		AttributeSchema,
		FloatValueSchema
	} from '$lib/api/v1/clients/client_service_pb';
	import { type Attribute, Unit } from '$lib/api/v1/clients/client_service_pb';
	import { ServiceAction } from '$lib/components/wh/service';
	import { create, toJsonString } from '@bufbuild/protobuf';
	import { BoolContent, FloatContent } from '$lib/components/wh/attributes';
	import { toHeadlineCase } from '$lib/tools/headline-case';

	let {
		others,
		serviceAction
	}: {
		others: Attribute[];
		serviceAction: ServiceAction;
	} = $props();

	let sendActionBool = async (id: string, val: boolean) => {
		serviceAction.send([
			create(ValueSchema, {
				id: id,
				bool: create(BoolValueSchema, {
					value: val
				})
			})
		]);
	};

	let sendActionFloat = async (id: string, val: number) => {
		serviceAction.send([
			create(ValueSchema, {
				id: id,
				float: create(FloatValueSchema, {
					value: val
				})
			})
		]);
	};

	let floatUnits = (unit: Unit) => {
		switch (unit) {
			case Unit.UNDEFINED:
				return '?';
			case Unit.PERCENTAGE:
				return '%';
			case Unit.ARC_DEGREES:
				return '°';
			case Unit.CELSIUS:
				return '°C';
			case Unit.LUX:
				return 'LUX';
			case Unit.SECONDS:
				return 's';
			case Unit.PPM:
				return 'PPM';
			case Unit.MICROGRAMS_PER_CUBIC_METER:
				return 'mg/m³';
			case Unit.VOLTS:
				return 'V';
			case Unit.AMPS:
				return 'A';
			case Unit.WATTS:
				return 'W';
			case Unit.MIREDS:
				return 'Mireds';
			case Unit.HECTOPASCAL:
				return 'hPa';
		}
		return '?';
	};
</script>

{#if others.length > 0}
	<div class="grid grid-cols-[auto_1fr_auto] pt-4 gap-4 items-center">
		<div class="col-span-3 text-muted-foreground border-t-2 font-semibold"></div>
		<!-- <div class="col-span-3 text-muted-foreground pt-3 border-t-2 font-semibold">Others</div> -->
		{#each others as other}
			{#if other.bool}
				<BoolContent
					name={toHeadlineCase(other.id)}
					attr={other.bool}
					onaction={(val) => sendActionBool(other.id, val)}
				/>
				<!-- {:else if other.color} -->
				<!-- {:else if other.duration} -->
				<!-- {:else if other.enum} -->
			{:else if other.float}
				<FloatContent
					name={toHeadlineCase(other.id)}
					value={other.float.value}
					min={other.float.min}
					max={other.float.max}
					onaction={(val) => sendActionFloat(other.id, val)}
					units={floatUnits(other.float.unit)}
				/>
				<!-- {:else if other.int} -->
				<!-- {:else if other.text} -->
				<!-- {:else if other.time} -->
			{:else}
				<div class="col-span-3 font-mono bg-muted px-4 py-2 rounded-md whitespace-pre overflow-x-auto">
					{toJsonString(AttributeSchema, other)}
				</div>
			{/if}
		{/each}
	</div>
{/if}
