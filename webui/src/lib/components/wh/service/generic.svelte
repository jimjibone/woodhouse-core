<script lang="ts">
	import {
		AttributeSchema,
		BoolValueSchema,
		DurationValueSchema,
		EnumValueSchema,
		FloatValueSchema,
		IntValueSchema,
		Permissions,
		ValueSchema,
		type Attribute,
		type TimeAttribute
	} from '$lib/api/v1/clients/client_service_pb';
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { SlidersHorizontalIcon } from '@lucide/svelte';
	import { create, toJsonString } from '@bufbuild/protobuf';
	import { BoolContent, DurationContent, EnumContent, FloatContent, IntContent } from '$lib/components/wh/attributes';
	import { toHeadlineCase } from '$lib/tools/headline-case';
	import { toPreciseDuration } from '$lib/tools/duration';
	import { unitLabel } from '$lib/tools/units';

	let { deviceID, service, ...rest }: StandardProps = $props();

	let serviceAction = new ServiceAction(deviceID, service.id);

	const isWritable = (perms: Permissions | undefined) =>
		perms === Permissions.PERM_READWRITE || perms === Permissions.PERM_WRITEONLY;

	const attrWritable = (attr: Attribute) =>
		isWritable(
			attr.bool?.perms ??
				attr.int?.perms ??
				attr.float?.perms ??
				attr.text?.perms ??
				attr.duration?.perms ??
				attr.time?.perms ??
				attr.color?.perms ??
				attr.enum?.perms
		);

	// Attributes arrive in no particular order, so sort them to keep the
	// display stable: writable controls first, then read-only values, each
	// group ordered by ID.
	let attrs = $derived(
		[...service.attrs].sort((a, b) => {
			const aWritable = attrWritable(a) ? 0 : 1;
			const bWritable = attrWritable(b) ? 0 : 1;
			if (aWritable !== bWritable) return aWritable - bWritable;
			return a.id < b.id ? -1 : a.id > b.id ? 1 : 0;
		})
	);

	const timeIsSet = (t: TimeAttribute) => t.seconds !== 0n || t.nanos !== 0;
	const timeDate = (t: TimeAttribute) => new Date(Number(t.seconds) * 1000 + t.nanos / 1000000);

	// Summarises an attribute value for the condensed details line. Returns an
	// empty string for zero or unset values so they don't clutter the summary
	// (they are still shown in the popup).
	const attrSummary = (attr: Attribute): string => {
		if (attr.bool) return attr.bool.value ? 'On' : 'Off';
		if (attr.int) return attr.int.value !== 0n ? attr.int.value.toLocaleString() + unitLabel(attr.int.unit) : '';
		if (attr.float)
			return attr.float.value !== 0
				? attr.float.value.toLocaleString(undefined, { maximumFractionDigits: 1 }) + unitLabel(attr.float.unit)
				: '';
		if (attr.text) return attr.text.value;
		if (attr.duration) return attr.duration.value !== 0n ? toPreciseDuration(Number(attr.duration.value)) : '';
		if (attr.time)
			return timeIsSet(attr.time)
				? timeDate(attr.time).toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' })
				: '';
		if (attr.enum) return attr.enum.value;
		if (attr.color) return attr.color.hueSat ? `Hue ${attr.color.hueSat.hue.toFixed(0)}°` : '';
		return '';
	};

	const maxSummaryValues = 3;
	let summaryValues = $derived(attrs.map(attrSummary).filter((val) => val !== ''));
	let shownValues = $derived(summaryValues.slice(0, maxSummaryValues));
	let moreCount = $derived(summaryValues.length - shownValues.length);

	const sendActionBool = async (id: string, val: boolean) => {
		serviceAction.send([create(ValueSchema, { id: id, bool: create(BoolValueSchema, { value: val }) })]);
	};

	const sendActionInt = async (id: string, val: bigint) => {
		serviceAction.send([create(ValueSchema, { id: id, int: create(IntValueSchema, { value: val }) })]);
	};

	const sendActionFloat = async (id: string, val: number) => {
		serviceAction.send([create(ValueSchema, { id: id, float: create(FloatValueSchema, { value: val }) })]);
	};

	const sendActionDuration = async (id: string, val: bigint) => {
		serviceAction.send([create(ValueSchema, { id: id, duration: create(DurationValueSchema, { value: val }) })]);
	};

	const sendActionEnum = async (id: string, val: string) => {
		serviceAction.send([create(ValueSchema, { id: id, enum: create(EnumValueSchema, { value: val }) })]);
	};
</script>

{#snippet icon()}
	<SlidersHorizontalIcon />
{/snippet}

{#snippet details()}
	{#each shownValues as val, i}
		<p class={i > 0 ? 'text-muted-foreground' : undefined}>{val}</p>
	{/each}
	{#if moreCount > 0}
		<p class="text-muted-foreground">+{moreCount}</p>
	{/if}
{/snippet}

{#snippet readRow(id: string, value: string)}
	<div>{toHeadlineCase(id)}</div>
	<div class="col-span-2">{value}</div>
{/snippet}

<ServiceRoot
	{deviceID}
	{...rest}
	{service}
	fallbackAlias={service.id}
	actionPending={serviceAction.pending}
	errorSignal={serviceAction.error}
	{icon}
	{details}
>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#each attrs as attr (attr.id)}
			{#if attr.bool}
				{#if isWritable(attr.bool.perms)}
					<BoolContent name={toHeadlineCase(attr.id)} attr={attr.bool} onaction={(val) => sendActionBool(attr.id, val)} />
				{:else}
					{@render readRow(attr.id, attr.bool.value ? 'On' : 'Off')}
				{/if}
			{:else if attr.int}
				{#if isWritable(attr.int.perms)}
					<IntContent
						name={toHeadlineCase(attr.id)}
						attr={attr.int}
						onaction={(val) => sendActionInt(attr.id, val)}
						units={unitLabel(attr.int.unit)}
					/>
				{:else}
					{@render readRow(attr.id, attr.int.value.toLocaleString() + unitLabel(attr.int.unit))}
				{/if}
			{:else if attr.float}
				{#if isWritable(attr.float.perms)}
					<FloatContent
						name={toHeadlineCase(attr.id)}
						value={attr.float.value}
						min={attr.float.min}
						max={attr.float.max}
						step={attr.float.step > 0 ? attr.float.step : 1}
						onaction={(val) => sendActionFloat(attr.id, val)}
						units={unitLabel(attr.float.unit)}
					/>
				{:else}
					{@render readRow(
						attr.id,
						attr.float.value.toLocaleString(undefined, { maximumFractionDigits: 1 }) + unitLabel(attr.float.unit)
					)}
				{/if}
			{:else if attr.duration}
				{#if isWritable(attr.duration.perms)}
					<DurationContent
						name={toHeadlineCase(attr.id)}
						attr={attr.duration}
						onaction={(val) => sendActionDuration(attr.id, val)}
					/>
				{:else}
					{@render readRow(attr.id, toPreciseDuration(Number(attr.duration.value)))}
				{/if}
			{:else if attr.enum}
				<EnumContent
					name={toHeadlineCase(attr.id)}
					attr={attr.enum}
					onaction={isWritable(attr.enum.perms) ? (val) => sendActionEnum(attr.id, val) : undefined}
				/>
			{:else if attr.text}
				{@render readRow(attr.id, attr.text.value)}
			{:else if attr.time}
				{@render readRow(attr.id, timeIsSet(attr.time) ? timeDate(attr.time).toLocaleString() : 'Not set')}
			{:else}
				<div class="col-span-3 font-mono bg-muted px-4 py-2 rounded-md whitespace-pre overflow-x-auto">
					{toJsonString(AttributeSchema, attr)}
				</div>
			{/if}
		{/each}
	</div>
</ServiceRoot>
