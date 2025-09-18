<script lang="ts">
	import { ValueSchema, IntValueSchema, EnumValueSchema } from '$lib/api/v1/clients/client_service_pb';
	import type { Attribute, EnumAttribute, IntAttribute } from '$lib/api/v1/clients/client_service_pb';
	import ServiceRoot, { type StandardProps } from "./service-root.svelte";
	import ServiceAction from './service-action.svelte';
	import { BlindsIcon } from '@lucide/svelte';
	import { create } from '@bufbuild/protobuf';
	import { IntContent, EnumContent, OthersContent } from '$lib/components/wh/attributes';

	let {
		deviceID,
		service,
		...rest
	}: StandardProps = $props();

	let attrPosition: IntAttribute | undefined = $state(undefined);
	let attrState: EnumAttribute | undefined = $state(undefined);
	let attrOthers: Attribute[] = $state([]);

	$effect(() => {
		let others: Attribute[] = [];
		for (const attr of service.attrs) {
			if (attr.id === 'position') {
				attrPosition = attr.int;
			} else if (attr.id === 'state') {
				attrState = attr.enum;
			} else {
				others = [...others, attr];
			}
		}
		attrOthers = others;
	});

	let serviceAction = new ServiceAction(deviceID, service.id);

	let sendActionPosition = async (val: bigint) => {
		serviceAction.send([
			create(ValueSchema, {
				id: 'position',
				int: create(IntValueSchema, {
					value: val
				})
			})
		]);
	};
	let sendActionState = async (val: string) => {
		serviceAction.send([
			create(ValueSchema, {
				id: 'state',
				enum: create(EnumValueSchema, {
					value: val
				})
			})
		]);
	};
</script>

{#snippet icon()}
	<BlindsIcon/>
{/snippet}

{#snippet details()}
	{#if attrPosition !== undefined}
		<!-- <GaugeIcon class="size-5"/> -->
		<p>
			{attrPosition.value.toLocaleString(undefined, { maximumFractionDigits: 1, minimumFractionDigits: 1 })}%
		</p>
	{/if}
	{#if attrState !== undefined}
		<p class="whitespace-pre text-muted-foreground">
			{#if attrState.value === ""}
				No state
			{:else}
				{attrState.value}
			{/if}
		</p>
	{/if}
{/snippet}

<ServiceRoot
	{deviceID}
	{...rest}
	service={service}
	actionPending={serviceAction.pending}
	errorSignal={serviceAction.error}
	icon={icon}
	details={details}
>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if attrPosition !== undefined}
			<IntContent
				name="Position"
				attr={attrPosition}
				onaction={sendActionPosition}
				units="%"
			/>
		{/if}
		{#if attrState !== undefined}
			<EnumContent
				name="State"
				attr={attrState}
				onaction={sendActionState}
			/>
		{/if}
		<OthersContent others={attrOthers} {serviceAction}/>
	</div>
</ServiceRoot>
