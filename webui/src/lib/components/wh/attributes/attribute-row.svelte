<script lang="ts">
	// One attribute of a service, rendered from its spec. Writable (RW)
	// attributes get a control that writes through the service action, others
	// are shown read-only. Emits grid cells for a
	// `grid grid-cols-[auto_1fr_auto]` parent.
	import type { Attribute } from '$lib/api/v2/clients/client_service_pb';
	import type ServiceAction from '$lib/components/wh/service/service-action.svelte';
	import {
		asColor,
		asDuration,
		asEnum,
		asFloat,
		asInt,
		asNumber,
		asText,
		boolValue,
		colorHueSatValue,
		durationValue,
		enumValue,
		floatValue,
		formatValue,
		intValue,
		isWritable,
		specEnumOptions,
		specKind,
		specRange,
		specUnit,
		textValue,
		valueKind,
		asBool
	} from '$lib/device';
	import { toHeadlineCase } from '$lib/tools/headline-case';
	import { unitLabel } from '$lib/tools/units';
	import { ImagesStore, imageKey } from '$lib/stores/images-stream';
	import BoolContent from './bool.svelte';
	import DurationContent from './duration.svelte';
	import EnumContent from './enum.svelte';
	import FloatContent from './float.svelte';
	import IntContent from './int.svelte';
	import NumberContent from './number.svelte';
	import TextContent from './text.svelte';
	import ValueView from './value-view.svelte';

	let {
		attr,
		deviceID,
		endpointID,
		serviceID,
		action
	}: {
		attr: Attribute;
		deviceID: string;
		endpointID: string;
		serviceID: string;
		action: ServiceAction;
	} = $props();

	const name = $derived(toHeadlineCase(attr.id));
	const writable = $derived(isWritable(attr));
	// The spec says what the attribute is; fall back to the value's own kind
	// if a spec was not sent.
	const kind = $derived(specKind(attr.spec) ?? valueKind(attr.value));
	const range = $derived(specRange(attr.spec));
	const units = $derived(unitLabel(specUnit(attr.spec)));
	const text = $derived(formatValue(attr.value, attr.spec));

	const hasBounds = $derived(range.min !== undefined && range.max !== undefined);
	const floatDigits = $derived(
		range.step !== undefined && range.step > 0 && range.step < 1
			? Math.min(4, Math.ceil(-Math.log10(range.step)))
			: 0
	);

	const color = $derived(asColor(attr.value));
	const canEditColor = $derived(
		writable && attr.spec?.kind.case === 'color' && attr.spec.kind.value.hueSat === true
	);

	const imageURL = $derived($ImagesStore.images.get(imageKey(deviceID, endpointID, serviceID, attr.id)));
</script>

{#snippet readRow(value: string)}
	<div>{name}</div>
	<div class="col-span-2">{value === '' ? '-' : value}</div>
{/snippet}

{#if kind === 'bool'}
	{#if writable}
		<BoolContent {name} value={asBool(attr.value)} onaction={(v) => action.writeValue(attr.id, boolValue(v))} />
	{:else}
		{@render readRow(text)}
	{/if}
{:else if kind === 'int'}
	{#if writable && hasBounds}
		<IntContent
			{name}
			value={asInt(attr.value)}
			min={BigInt(range.min!)}
			max={BigInt(range.max!)}
			step={BigInt(Math.max(1, range.step ?? 1))}
			{units}
			onaction={(v) => action.writeValue(attr.id, intValue(v))}
		/>
	{:else if writable}
		<NumberContent
			{name}
			value={asNumber(attr.value)}
			min={range.min}
			max={range.max}
			{units}
			onaction={(v) => action.writeValue(attr.id, intValue(v))}
		/>
	{:else}
		{@render readRow(text)}
	{/if}
{:else if kind === 'float'}
	{#if writable && hasBounds}
		<FloatContent
			{name}
			value={asFloat(attr.value)}
			min={range.min!}
			max={range.max!}
			step={range.step && range.step > 0 ? range.step : 1}
			maximumFractionDigits={floatDigits}
			{units}
			onaction={(v) => action.writeValue(attr.id, floatValue(v))}
		/>
	{:else if writable}
		<NumberContent
			{name}
			value={asNumber(attr.value)}
			min={range.min}
			max={range.max}
			{units}
			onaction={(v) => action.writeValue(attr.id, floatValue(v))}
		/>
	{:else}
		{@render readRow(text)}
	{/if}
{:else if kind === 'duration'}
	{#if writable && hasBounds}
		<DurationContent
			{name}
			value={asDuration(attr.value)}
			min={BigInt(range.min!)}
			max={BigInt(range.max!)}
			step={BigInt(Math.max(1, range.step ?? 1))}
			onaction={(v) => action.writeValue(attr.id, durationValue(v))}
		/>
	{:else if writable}
		<NumberContent
			{name}
			value={asNumber(attr.value)}
			min={range.min}
			max={range.max}
			units="ms"
			onaction={(v) => action.writeValue(attr.id, durationValue(v))}
		/>
	{:else}
		{@render readRow(text)}
	{/if}
{:else if kind === 'text'}
	{#if writable}
		<TextContent {name} value={asText(attr.value)} onaction={(v) => action.writeValue(attr.id, textValue(v))} />
	{:else}
		{@render readRow(text)}
	{/if}
{:else if kind === 'enum'}
	<EnumContent
		{name}
		value={asEnum(attr.value)}
		options={specEnumOptions(attr.spec)}
		onaction={writable ? (v) => action.writeValue(attr.id, enumValue(v)) : undefined}
	/>
{:else if kind === 'color'}
	{#if canEditColor}
		<!-- Hue and saturation sliders; each sends the whole color. -->
		<FloatContent
			name={name + ' Hue'}
			value={color?.hueSat?.hue}
			min={0}
			max={360}
			units=" deg"
			onaction={(h) => action.writeValue(attr.id, colorHueSatValue(h, color?.hueSat?.sat ?? 100))}
		/>
		<FloatContent
			name={name + ' Saturation'}
			value={color?.hueSat?.sat}
			min={0}
			max={100}
			units="%"
			onaction={(s) => action.writeValue(attr.id, colorHueSatValue(color?.hueSat?.hue ?? 0, s))}
		/>
	{:else}
		<div>{name}</div>
		<div class="col-span-2"><ValueView value={attr.value} spec={attr.spec} /></div>
	{/if}
{:else if kind === 'image'}
	<div>{name}</div>
	<div class="col-span-2">
		{#if imageURL}
			<img src={imageURL.url} alt={name} class="max-h-64 rounded-md" />
		{:else}
			<span class="text-muted-foreground">No image</span>
		{/if}
	</div>
{:else if kind === 'list' || kind === 'struct'}
	<div class="self-start">{name}</div>
	<div class="col-span-2 min-w-0"><ValueView value={attr.value} spec={attr.spec} /></div>
{:else}
	{@render readRow(text)}
{/if}
