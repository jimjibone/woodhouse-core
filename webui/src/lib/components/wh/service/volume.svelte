<script lang="ts">
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { MinusIcon, PlusIcon, Volume2Icon, VolumeXIcon } from '@lucide/svelte';
	import { Button } from '$lib/components/ui/button';
	import { BoolContent, IntContent, OthersContent } from '$lib/components/wh/attributes';
	import {
		attrBool,
		attrInt,
		boolValue,
		field,
		findAttr,
		findCommand,
		intValue,
		isWritable,
		specRange
	} from '$lib/device';
	import { Volume } from '$lib/schema/zz_services';

	let { deviceID, endpointID, service, ...rest }: StandardProps = $props();

	// How far the + and - buttons move the volume.
	const STEP_DELTA = 5;

	const level = $derived(attrInt(service, Volume.attributes.level));
	const muted = $derived(attrBool(service, Volume.attributes.muted));
	const groupLevel = $derived(attrInt(service, Volume.attributes.groupLevel));
	const groupMuted = $derived(attrBool(service, Volume.attributes.groupMuted));
	const levelAttr = $derived(findAttr(service, Volume.attributes.level));
	const mutedAttr = $derived(findAttr(service, Volume.attributes.muted));
	const groupLevelAttr = $derived(findAttr(service, Volume.attributes.groupLevel));
	const groupMutedAttr = $derived(findAttr(service, Volume.attributes.groupMuted));
	const canStep = $derived(findCommand(service, Volume.commands.step) !== undefined);

	const bounds = (spec: typeof levelAttr) => {
		const r = specRange(spec?.spec);
		return {
			min: BigInt(r.min ?? 0),
			max: BigInt(r.max ?? 100),
			step: BigInt(Math.max(1, r.step ?? 1))
		};
	};
	const levelBounds = $derived(bounds(levelAttr));
	const groupLevelBounds = $derived(bounds(groupLevelAttr));

	// svelte-ignore state_referenced_locally
	const serviceAction = new ServiceAction(deviceID, endpointID, service.id);

	const sendLevel = (val: bigint) => serviceAction.writeValue(Volume.attributes.level, intValue(val));
	const sendMuted = (val: boolean) => serviceAction.writeValue(Volume.attributes.muted, boolValue(val));
	const sendGroupLevel = (val: bigint) => serviceAction.writeValue(Volume.attributes.groupLevel, intValue(val));
	const sendGroupMuted = (val: boolean) => serviceAction.writeValue(Volume.attributes.groupMuted, boolValue(val));
	const step = (delta: number) => serviceAction.command(Volume.commands.step, [field('delta', intValue(delta))]);
</script>

{#snippet icon()}
	{#if muted}
		<VolumeXIcon />
	{:else}
		<Volume2Icon />
	{/if}
{/snippet}

{#snippet details()}
	{#if level !== undefined}
		<p>{level.toLocaleString(undefined, { maximumFractionDigits: 0 })}%</p>
	{/if}
	{#if muted}
		<p class="text-muted-foreground">Muted</p>
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
		{#if levelAttr !== undefined}
			{#if isWritable(levelAttr)}
				<IntContent
					name="Volume"
					value={level}
					min={levelBounds.min}
					max={levelBounds.max}
					step={levelBounds.step}
					onaction={sendLevel}
					units="%"
				/>
			{:else}
				<div>Volume</div>
				<div class="col-span-2">{level === undefined ? '-' : level.toString() + '%'}</div>
			{/if}
		{/if}
		{#if mutedAttr !== undefined}
			{#if isWritable(mutedAttr)}
				<BoolContent name="Muted" value={muted} onaction={sendMuted} />
			{:else}
				<div>Muted</div>
				<div class="col-span-2">{muted === undefined ? '-' : muted ? 'Yes' : 'No'}</div>
			{/if}
		{/if}
		{#if groupLevelAttr !== undefined}
			{#if isWritable(groupLevelAttr)}
				<IntContent
					name="Group Volume"
					value={groupLevel}
					min={groupLevelBounds.min}
					max={groupLevelBounds.max}
					step={groupLevelBounds.step}
					onaction={sendGroupLevel}
					units="%"
				/>
			{:else}
				<div>Group Volume</div>
				<div class="col-span-2">{groupLevel === undefined ? '-' : groupLevel.toString() + '%'}</div>
			{/if}
		{/if}
		{#if groupMutedAttr !== undefined}
			{#if isWritable(groupMutedAttr)}
				<BoolContent name="Group Muted" value={groupMuted} onaction={sendGroupMuted} />
			{:else}
				<div>Group Muted</div>
				<div class="col-span-2">{groupMuted === undefined ? '-' : groupMuted ? 'Yes' : 'No'}</div>
			{/if}
		{/if}
	</div>

	{#if canStep}
		<div class="flex flex-row justify-center gap-2 pt-4">
			<Button
				variant="secondary"
				size="icon"
				class="cursor-pointer"
				aria-label="Volume down"
				onclick={() => step(-STEP_DELTA)}
			>
				<MinusIcon />
			</Button>
			<Button
				variant="secondary"
				size="icon"
				class="cursor-pointer"
				aria-label="Volume up"
				onclick={() => step(STEP_DELTA)}
			>
				<PlusIcon />
			</Button>
		</div>
	{/if}

	<OthersContent
		{deviceID}
		{endpointID}
		{service}
		action={serviceAction}
		exclude={[
			Volume.attributes.level,
			Volume.attributes.muted,
			Volume.attributes.groupLevel,
			Volume.attributes.groupMuted
		]}
		excludeCommands={[Volume.commands.step]}
	/>
</ServiceRoot>
