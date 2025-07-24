<script lang="ts">
	import { ValueSchema, EnumValueSchema } from '$lib/api/v1/clients/client_service_pb';
	import type { Attribute, EnumAttribute, Service } from '$lib/api/v1/clients/client_service_pb';
	import ServiceRoot, { type StandardProps } from "./service-root.svelte";
	import ServiceAction from './service-action.svelte';
	import { Rows3Icon } from '@lucide/svelte';
	import { create } from '@bufbuild/protobuf';
	import { EnumContent, OthersContent } from '$lib/components/wh/attributes';

	let {
		deviceID,
		service,
		...rest
	}: StandardProps = $props();

	let attrValue: EnumAttribute | undefined = $state(undefined);
	let attrOthers: Attribute[] = $state([]);

	$effect(() => {
		let others: Attribute[] = [];
		for (const attr of service.attrs) {
			if (attr.id === 'value') {
				attrValue = attr.enum;
			} else {
				others = [...others, attr];
			}
		}
		attrOthers = others;
	});

	let serviceAction = new ServiceAction(deviceID, service.id);

	let sendActionValue = async (val: string) => {
		serviceAction.send([
			create(ValueSchema, {
				id: 'value',
				enum: create(EnumValueSchema, {
					value: val
				})
			})
		]);
	};
</script>

{#snippet icon()}
	<Rows3Icon/>
{/snippet}

{#snippet details()}
	{#if attrValue !== undefined}
		<p class="whitespace-pre">
			{#if attrValue.value === ""}
				No state
			{:else}
				{attrValue.value}
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
		{#if attrValue !== undefined}
			<EnumContent
				name="Value"
				attr={attrValue}
				onaction={sendActionValue}
			/>
		{/if}
		<OthersContent others={attrOthers} {serviceAction}/>
	</div>
</ServiceRoot>
