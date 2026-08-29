<script lang="ts">
	import type { Zone } from '$lib/api/v1/clients/zone_pb';
	import Button from '$lib/components/ui/button/button.svelte';
	import * as Field from '$lib/components/ui/field/index.js';
	import Input from '$lib/components/ui/input/input.svelte';
	import { AddZone, UpdateZone } from '$lib/stores/requests';
	import type { ConnectError } from '@connectrpc/connect';
	import { toSentenceCase } from '$lib/tools/headline-case';
	import { DevicesStore, type DevicesStoreDevice } from '$lib/stores/devices-stream';
	import { ZonesStore } from '$lib/stores/zones-stream';
	import { ZONE_ICONS, DEFAULT_ZONE_ICON_KEY } from '$lib/zone-icons';
	import { cn } from '$lib/utils';
	import { onDestroy } from 'svelte';

	let { zone = undefined, onSuccess }: { zone?: Zone; onSuccess: () => void } = $props();

	const id = $props.id();

	const isEditing = $derived(zone !== undefined);

	// Captured once from the prop - these are the form's initial values, not a
	// live mirror of the zone.
	// eslint-disable-next-line svelte/valid-state-references
	let selectedIcon = $state<string>(zone?.icon ?? DEFAULT_ZONE_ICON_KEY);
	let selectedDevices = $state<Set<string>>(new Set(zone?.deviceIds ?? []));

	let devices = $state<DevicesStoreDevice[]>([]);
	let zones = $state<Zone[]>([]);
	onDestroy(DevicesStore.subscribe((s) => (devices = s.devices)));
	onDestroy(ZonesStore.subscribe((s) => (zones = s.zones)));

	const availableDevices = $derived(
		[...devices].sort((a, b) => {
			const an = a.name ?? a.id;
			const bn = b.name ?? b.id;
			return an > bn ? 1 : bn > an ? -1 : 0;
		})
	);

	// A device lives in exactly one zone, so ticking one here takes it off
	// whichever zone holds it now. Naming that zone makes the move visible
	// before it happens rather than after.
	const otherZoneNames = $derived.by(() => {
		const names = new Map<string, string>();
		for (const z of zones) {
			if (z.id === zone?.id) continue;
			for (const deviceID of z.deviceIds) {
				names.set(deviceID, z.name);
			}
		}
		return names;
	});

	// The picker scrolls, and the current icon can sit well down the list - on
	// the new-zone form the default is already below the fold. Bring it into
	// view on open so the form always shows what is selected.
	let iconGrid = $state<HTMLDivElement | null>(null);
	$effect(() => {
		iconGrid?.querySelector('[aria-pressed="true"]')?.scrollIntoView({ block: 'nearest' });
	});

	function toggleDevice(deviceID: string) {
		const next = new Set(selectedDevices);
		if (next.has(deviceID)) {
			next.delete(deviceID);
		} else {
			next.add(deviceID);
		}
		selectedDevices = next;
	}

	let error = $state<string | null>(null);
	let submitting = $state(false);

	async function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		const form = event.target as HTMLFormElement;
		const data = new FormData(form);
		const name = data.get(`name-${id}`)?.toString().trim();

		if (!name) {
			error = 'name is required';
			return;
		}

		const deviceIds = Array.from(selectedDevices);

		error = null;
		submitting = true;

		let res: ConnectError | null;

		if (isEditing) {
			// Only send what changed; deviceIds is always sent so clearing the
			// zone is expressible.
			res = await UpdateZone(zone!.id, {
				name: name !== zone!.name ? name : undefined,
				icon: selectedIcon !== zone!.icon ? selectedIcon : undefined,
				deviceIds
			});
		} else {
			res = await AddZone(name, selectedIcon, deviceIds);
		}

		submitting = false;

		if (res) {
			error = toSentenceCase(res.rawMessage);
		} else {
			onSuccess();
		}
	}
</script>

<form onsubmit={handleSubmit}>
	<Field.Set>
		<Field.Legend>{isEditing ? 'Edit Zone' : 'New Zone'}</Field.Legend>
		<Field.Description>
			{isEditing ? 'Update the zone name, icon and devices.' : 'Create a room or area to group devices by.'}
		</Field.Description>

		<Field.Group>
			<Field.Field>
				<Field.Label for="name-{id}">Name</Field.Label>
				<Input
					id="name-{id}"
					name="name-{id}"
					type="text"
					placeholder="Kitchen"
					autocomplete="off"
					value={zone?.name ?? ''}
					required
				/>
				<Field.Description>Shown in the side nav and the mobile nav.</Field.Description>
			</Field.Field>
		</Field.Group>

		<Field.Group>
			<Field.Set>
				<Field.Label>Icon</Field.Label>
				<Field.Description>Appears beside the zone name in the navigation.</Field.Description>
				<div
					bind:this={iconGrid}
					class="grid grid-cols-8 gap-1 max-h-40 overflow-y-auto rounded-md border bg-background p-2"
				>
					{#each ZONE_ICONS as option (option.key)}
						{@const OptionIcon = option.icon}
						<button
							type="button"
							title={option.label}
							aria-label={option.label}
							aria-pressed={selectedIcon === option.key}
							onclick={() => (selectedIcon = option.key)}
							class={cn(
								'flex aspect-square items-center justify-center rounded-md hover:bg-accent cursor-pointer',
								selectedIcon === option.key && 'bg-primary text-primary-foreground hover:bg-primary'
							)}
						>
							<OptionIcon class="size-5" />
						</button>
					{/each}
				</div>
			</Field.Set>
		</Field.Group>

		<Field.Group>
			<Field.Set>
				<Field.Label>Devices</Field.Label>
				<Field.Description>Select the devices in this zone.</Field.Description>
				<div class="flex flex-col gap-0.5 max-h-52 overflow-y-auto rounded-md border bg-background p-1">
					{#if availableDevices.length === 0}
						<p class="text-muted-foreground text-sm py-3 text-center">No devices found.</p>
					{/if}
					{#each availableDevices as device (device.id)}
						{@const otherZone = otherZoneNames.get(device.id)}
						<label
							class="flex items-center gap-2.5 rounded-sm px-2 py-1.5 hover:bg-accent cursor-pointer text-sm select-none"
						>
							<input
								type="checkbox"
								checked={selectedDevices.has(device.id)}
								onchange={() => toggleDevice(device.id)}
								class="size-4 rounded accent-primary cursor-pointer"
							/>
							<span class={cn('flex-1 font-medium truncate', !device.name && 'font-mono')}>
								{device.name ?? device.id}
							</span>
							{#if otherZone}
								<span class="text-muted-foreground text-xs shrink-0">
									{selectedDevices.has(device.id) ? 'moving from' : 'in'}
									{otherZone}
								</span>
							{/if}
						</label>
					{/each}
				</div>
				<Field.Description>
					{selectedDevices.size} device{selectedDevices.size === 1 ? '' : 's'} selected.
				</Field.Description>
			</Field.Set>
		</Field.Group>

		{#if error}
			<Field.Error>{error}</Field.Error>
		{/if}

		<Field.Field>
			<Button type="submit" class="cursor-pointer" disabled={submitting}>
				{isEditing ? 'Save Changes' : 'Create Zone'}
			</Button>
		</Field.Field>
	</Field.Set>
</form>
