<script lang="ts">
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { BlindsIcon } from '@lucide/svelte';
	import { IntContent, EnumContent, OthersContent } from '$lib/components/wh/attributes';
	import {
		attrEnum,
		attrInt,
		field,
		findAttr,
		findCommand,
		intValue,
		isWritable,
		specEnumOptions,
		specRange
	} from '$lib/device';
	import { Cover } from '$lib/schema/zz_services';

	let { deviceID, endpointID, service, ...rest }: StandardProps = $props();

	const position = $derived(attrInt(service, Cover.attributes.position));
	const tilt = $derived(attrInt(service, Cover.attributes.tilt));
	const coverState = $derived(attrEnum(service, Cover.attributes.state));
	const positionAttr = $derived(findAttr(service, Cover.attributes.position));
	const tiltAttr = $derived(findAttr(service, Cover.attributes.tilt));
	const stateAttr = $derived(findAttr(service, Cover.attributes.state));

	// A position is set by the go_to command, or by writing the attribute if
	// the bridge made it writable. The same goes for tilt and tilt_to.
	const canGoTo = $derived(findCommand(service, Cover.commands.goTo) !== undefined);
	const canTiltTo = $derived(findCommand(service, Cover.commands.tiltTo) !== undefined);

	const bounds = (spec: typeof positionAttr) => {
		const r = specRange(spec?.spec);
		return { min: BigInt(r.min ?? 0), max: BigInt(r.max ?? 100), step: BigInt(Math.max(1, r.step ?? 1)) };
	};
	const positionBounds = $derived(bounds(positionAttr));
	const tiltBounds = $derived(bounds(tiltAttr));

	let serviceAction = new ServiceAction(deviceID, endpointID, service.id);

	let sendPosition = (val: bigint) => {
		if (canGoTo) {
			serviceAction.command(Cover.commands.goTo, [field('position', intValue(val))]);
		} else {
			serviceAction.writeValue(Cover.attributes.position, intValue(val));
		}
	};
	let sendTilt = (val: bigint) => {
		if (canTiltTo) {
			serviceAction.command(Cover.commands.tiltTo, [field('tilt', intValue(val))]);
		} else {
			serviceAction.writeValue(Cover.attributes.tilt, intValue(val));
		}
	};
</script>

{#snippet icon()}
	<BlindsIcon />
{/snippet}

{#snippet details()}
	{#if position !== undefined}
		<p>
			{position.toLocaleString(undefined, { maximumFractionDigits: 0, minimumFractionDigits: 0 })}%
		</p>
	{/if}
	{#if stateAttr !== undefined}
		<p class="whitespace-pre text-muted-foreground">
			{#if !coverState}
				No state
			{:else}
				{coverState}
			{/if}
		</p>
	{/if}
{/snippet}

<ServiceRoot
	{deviceID}
	{endpointID}
	{...rest}
	{service}
	actionPending={serviceAction.pending}
	errorSignal={serviceAction.error}
	{icon}
	{details}
>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if positionAttr !== undefined}
			{#if canGoTo || isWritable(positionAttr)}
				<IntContent
					name="Position"
					value={position}
					min={positionBounds.min}
					max={positionBounds.max}
					step={positionBounds.step}
					units="%"
					onaction={sendPosition}
				/>
			{:else}
				<div>Position</div>
				<div class="col-span-2">{position === undefined ? '-' : position.toString() + '%'}</div>
			{/if}
		{/if}
		{#if tiltAttr !== undefined}
			{#if canTiltTo || isWritable(tiltAttr)}
				<IntContent
					name="Tilt"
					value={tilt}
					min={tiltBounds.min}
					max={tiltBounds.max}
					step={tiltBounds.step}
					units="%"
					onaction={sendTilt}
				/>
			{:else}
				<div>Tilt</div>
				<div class="col-span-2">{tilt === undefined ? '-' : tilt.toString() + '%'}</div>
			{/if}
		{/if}
		{#if stateAttr !== undefined}
			<EnumContent name="State" value={coverState} options={specEnumOptions(stateAttr.spec)} />
		{/if}
	</div>
	<OthersContent
		{deviceID}
		{endpointID}
		{service}
		action={serviceAction}
		exclude={[Cover.attributes.position, Cover.attributes.tilt, Cover.attributes.state]}
		excludeCommands={[Cover.commands.goTo, Cover.commands.tiltTo]}
		separator={false}
	/>
</ServiceRoot>
