<script lang="ts">
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { UserPlusIcon } from '@lucide/svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { OthersContent } from '$lib/components/wh/attributes';
	import TimeSince from '$lib/components/wh/ui/time-since.svelte';
	import {
		asEnum,
		asList,
		asStruct,
		asText,
		asTime,
		attrDuration,
		attrEnum,
		attrText,
		attrTime,
		durationValue,
		field,
		fieldValue,
		findAttr,
		findCommand,
		textValue
	} from '$lib/device';
	import { toPreciseDuration } from '$lib/tools/duration';
	import { toHeadlineCase } from '$lib/tools/headline-case';
	import { Pairing } from '$lib/schema/zz_services';

	let { deviceID, endpointID, service, ...rest }: StandardProps = $props();

	const pairingState = $derived(attrEnum(service, Pairing.attributes.state));
	const ends = $derived(attrTime(service, Pairing.attributes.ends));
	const remaining = $derived(attrDuration(service, Pairing.attributes.remaining));
	const lastError = $derived(attrText(service, Pairing.attributes.lastError));
	const open = $derived(pairingState === 'open');

	const canStart = $derived(findCommand(service, Pairing.commands.start) !== undefined);
	const canStop = $derived(findCommand(service, Pairing.commands.stop) !== undefined);
	const canSubmitCode = $derived(
		service.features.includes(Pairing.features.codeEntry) &&
			findCommand(service, Pairing.commands.submitCode) !== undefined
	);
	const hasDuration = $derived(
		findCommand(service, Pairing.commands.start)?.params.some((p) => p.id === 'duration') ?? false
	);

	// Recent attempts, newest first.
	type Recent = {
		deviceID: string;
		name: string;
		state: string;
		error: string;
		time: Date | undefined;
	};
	const recent: Recent[] = $derived(
		(asList(findAttr(service, Pairing.attributes.recent)?.value)?.items ?? []).map((item) => {
			const fields = asStruct(item)?.fields;
			return {
				deviceID: asText(fieldValue(fields, 'device_id')) ?? '',
				name: asText(fieldValue(fields, 'name')) ?? '',
				state: asEnum(fieldValue(fields, 'state')) ?? '',
				error: asText(fieldValue(fields, 'error')) ?? '',
				time: asTime(fieldValue(fields, 'time'))
			};
		})
	);

	// svelte-ignore state_referenced_locally
	const serviceAction = new ServiceAction(deviceID, endpointID, service.id);

	// An empty duration uses the device default.
	let durationText = $state('');
	const durationSecs = $derived(durationText.trim() === '' ? undefined : Number(durationText));
	const durationValid = $derived(
		durationSecs === undefined || (Number.isFinite(durationSecs) && durationSecs >= 0)
	);

	const start = () => {
		const params =
			hasDuration && durationSecs !== undefined
				? [field('duration', durationValue(Math.round(durationSecs * 1000)))]
				: [];
		serviceAction.command(Pairing.commands.start, params);
	};
	const stop = () => serviceAction.command(Pairing.commands.stop);

	let code = $state('');
	const submitCode = async () => {
		const trimmed = code.trim();
		if (trimmed === '') return;
		const result = await serviceAction.command(Pairing.commands.submitCode, [field('code', textValue(trimmed))]);
		if (result !== undefined) code = '';
	};

	let drawerOpen = $state(false);
</script>

{#snippet icon()}
	<UserPlusIcon />
{/snippet}

{#snippet details()}
	{#if pairingState !== undefined}
		<p>{toHeadlineCase(pairingState)}</p>
	{/if}
	{#if open && remaining !== undefined}
		<p class="text-muted-foreground">{toPreciseDuration(Number(remaining))}</p>
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
	iconclass={open ? 'bg-green-400 text-black' : pairingState === 'busy' ? 'bg-amber-400 text-black' : false}
	{details}
	bind:drawerOpen
>
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		<div>State</div>
		<div class="col-span-2">{pairingState === undefined ? '-' : toHeadlineCase(pairingState)}</div>
		{#if remaining !== undefined && open}
			<div>Remaining</div>
			<div class="col-span-2">{toPreciseDuration(Number(remaining))}</div>
		{/if}
		{#if ends !== undefined && open}
			<div>Ends</div>
			<div class="col-span-2">{ends.toLocaleTimeString()}</div>
		{/if}
		{#if lastError}
			<div>Last Error</div>
			<div class="col-span-2 text-error-foreground">{lastError}</div>
		{/if}
		{#if canStart && hasDuration}
			<div>Duration</div>
			<div class="col-span-2 flex flex-row items-center gap-2">
				<Input
					type="text"
					inputmode="numeric"
					bind:value={durationText}
					placeholder="Default"
					aria-invalid={!durationValid}
				/>
				<span class="text-muted-foreground">s</span>
			</div>
		{/if}
	</div>

	{#if canStart || canStop}
		<div class="flex flex-row gap-2 pt-4">
			{#if canStart}
				<Button class="cursor-pointer" disabled={!durationValid} onclick={start}>
					{open ? 'Restart' : 'Start'}
				</Button>
			{/if}
			{#if canStop}
				<Button variant="secondary" class="cursor-pointer" disabled={pairingState === 'idle'} onclick={stop}>
					Stop
				</Button>
			{/if}
		</div>
	{/if}

	{#if canSubmitCode}
		<form
			class="flex flex-row gap-2 pt-4"
			onsubmit={(event) => {
				event.preventDefault();
				submitCode();
			}}
		>
			<Input type="text" bind:value={code} placeholder="Setup code" autocomplete="off" />
			<Button type="submit" class="cursor-pointer" disabled={code.trim() === ''}>Submit Code</Button>
		</form>
	{/if}

	{#if recent.length > 0}
		<div class="pt-4 grid gap-2">
			<div class="text-muted-foreground border-t-2 font-semibold"></div>
			<div class="font-medium">Recent</div>
			<div class="overflow-x-auto rounded-md border">
				<table class="w-full text-sm">
					<thead class="bg-muted/50 text-left text-muted-foreground">
						<tr>
							<th class="px-2 py-1 font-medium whitespace-nowrap">Device</th>
							<th class="px-2 py-1 font-medium whitespace-nowrap">State</th>
							<th class="px-2 py-1 font-medium whitespace-nowrap">Error</th>
							<th class="px-2 py-1 font-medium whitespace-nowrap">Time</th>
						</tr>
					</thead>
					<tbody>
						{#each recent as entry, i (i)}
							<tr class="border-t">
								<td class="px-2 py-1 align-top">
									{#if entry.deviceID !== ''}
										<a
											class="underline underline-offset-2"
											href={'/devices/' + entry.deviceID}
											onclick={(event) => {
												event.stopPropagation();
												drawerOpen = false;
											}}
										>
											{entry.name !== '' ? entry.name : entry.deviceID}
										</a>
									{:else}
										{entry.name}
									{/if}
								</td>
								<td class="px-2 py-1 align-top">{toHeadlineCase(entry.state)}</td>
								<td class="px-2 py-1 align-top">{entry.error}</td>
								<td class="px-2 py-1 align-top whitespace-nowrap">
									{#if entry.time}
										<TimeSince past={entry.time} warn={false} />
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	{/if}

	<OthersContent
		{deviceID}
		{endpointID}
		{service}
		action={serviceAction}
		exclude={[
			Pairing.attributes.state,
			Pairing.attributes.ends,
			Pairing.attributes.remaining,
			Pairing.attributes.lastError,
			Pairing.attributes.recent
		]}
		excludeCommands={[Pairing.commands.start, Pairing.commands.stop, Pairing.commands.submitCode]}
	/>
</ServiceRoot>
