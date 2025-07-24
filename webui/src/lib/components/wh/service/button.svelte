<script lang="ts">
	import type { Attribute, DurationAttribute, EnumAttribute, Service } from '$lib/api/v1/clients/client_service_pb';
	import ServiceRoot, { type StandardProps } from "./service-root.svelte";
	import ServiceAction from './service-action.svelte';
	import { PointerIcon } from '@lucide/svelte';
	import { EnumContent, OthersContent } from '$lib/components/wh/attributes';

	let {
		deviceID,
		service,
		...rest
	}: StandardProps = $props();

	let attrState: EnumAttribute | undefined = $state(undefined);
	let attrDuration: DurationAttribute | undefined = $state(undefined);
	let attrOthers: Attribute[] = $state([]);

	$effect(() => {
		let others: Attribute[] = [];
		for (const attr of service.attrs) {
			if (attr.id === 'state') {
				attrState = attr.enum;
			} else if (attr.id === 'duration') {
				attrDuration = attr.duration;
			} else {
				others = [...others, attr];
			}
		}
		attrOthers = others;
	});

	let serviceAction = new ServiceAction(deviceID, service.id);
</script>

{#snippet icon()}
	<PointerIcon />
{/snippet}

{#snippet details()}
	{#if attrState !== undefined}
		{#if attrState.value === ""}
			<p>
				No state
			</p>
		{:else}
			<p>
				{attrState.value}
			</p>
		{/if}
	{/if}
	{#if attrDuration !== undefined}
		{#if attrDuration.value > 0}
			<p class="text-muted-foreground">
				{attrDuration.value.toLocaleString(undefined, { maximumFractionDigits: 0 })}ms
			</p>
		{/if}
	{/if}
{/snippet}

<ServiceRoot
	{deviceID}
	{...rest}
	service={service}
	icon={icon}
	details={details}
>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if attrState !== undefined}
			<EnumContent
				name="State"
				attr={attrState}
			/>
		{/if}
		{#if attrDuration !== undefined}
			{#if attrDuration.value > 0}
				<div>Duration</div>
				<div class="col-span-2">
					{attrDuration.value.toLocaleString(undefined, { maximumFractionDigits: 0 })}ms
				</div>
			{/if}
		{/if}
		<OthersContent others={attrOthers} {serviceAction}/>
	</div>
</ServiceRoot>
