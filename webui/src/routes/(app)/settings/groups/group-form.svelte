<script lang="ts">
	import type { Group } from '$lib/api/v2/clients/group_pb';
	import { Service_ServiceType } from '$lib/api/v2/clients/client_service_pb';
	import Button from '$lib/components/ui/button/button.svelte';
	import * as Field from '$lib/components/ui/field/index.js';
	import Input from '$lib/components/ui/input/input.svelte';
	import { AddGroup, UpdateGroup } from '$lib/stores/requests';
	import { GroupMemberSchema } from '$lib/api/v2/clients/group_pb';
	import { create } from '@bufbuild/protobuf';
	import type { ConnectError } from '@connectrpc/connect';
	import { toSentenceCase } from '$lib/tools/headline-case';
	import { DevicesStore, type DevicesStoreDevice } from '$lib/stores/devices-stream';
	import { onDestroy } from 'svelte';
	import { iterServices } from '$lib/device';
	import { keyOf, memberLabel, serviceTypeOptions, typeLabel } from './members';

	let { group = undefined, onSuccess }: { group?: Group; onSuccess: () => void } = $props();

	const id = $props.id();

	const isEditing = $derived(group !== undefined);

	// Capture initial values from prop – these are intentionally one-time initialisations.
	// eslint-disable-next-line svelte/valid-state-references
	let selectedType = $state<Service_ServiceType>(group?.type ?? Service_ServiceType.ON_OFF);

	// Track selected members as memberKey(device, endpoint, service) keys.
	// Initialised once from the group prop (intentional one-time capture).
	const initialMembers = group?.members.map(keyOf) ?? [];
	let selectedMembers = $state<Set<string>>(new Set(initialMembers));

	let devices = $state<DevicesStoreDevice[]>([]);
	onDestroy(DevicesStore.subscribe((s) => (devices = s.devices)));

	type MemberOption = {
		deviceId: string;
		endpointId: string;
		serviceId: string;
		label: string;
		key: string;
	};

	let availableMembers = $derived<MemberOption[]>(
		devices
			.flatMap((device) =>
				[...iterServices(device)]
					.filter(({ service }) => service.typ === selectedType)
					.map(({ endpoint, service }) => {
						const ref = { deviceId: device.id, endpointId: endpoint.id, serviceId: service.id };
						return { ...ref, label: memberLabel(devices, ref), key: keyOf(ref) };
					})
			)
			.sort((a, b) => (a.label > b.label ? 1 : b.label > a.label ? -1 : 0))
	);

	// When the type changes while creating, clear member selection
	$effect(() => {
		if (!isEditing) {
			selectedType;
			selectedMembers = new Set();
		}
	});

	function toggleMember(key: string) {
		const next = new Set(selectedMembers);
		if (next.has(key)) {
			next.delete(key);
		} else {
			next.add(key);
		}
		selectedMembers = next;
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

		const members = Array.from(selectedMembers).map((key) => {
			const [deviceId, endpointId, serviceId] = JSON.parse(key) as [string, string, string];
			return create(GroupMemberSchema, { deviceId, endpointId, serviceId });
		});

		error = null;
		submitting = true;

		let res: ConnectError | null;

		if (isEditing) {
			// Only send name if it changed; always send the full member list
			const updatedName = name !== group!.name ? name : undefined;
			res = await UpdateGroup(group!.id, updatedName, members);
		} else {
			res = await AddGroup(name, selectedType, members);
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
		<Field.Legend>{isEditing ? 'Edit Group' : 'New Group'}</Field.Legend>
		<Field.Description>
			{isEditing ? 'Update the group name and members.' : 'Create a new group of devices.'}
		</Field.Description>

		<Field.Group>
			<Field.Field>
				<Field.Label for="name-{id}">Name</Field.Label>
				<Input
					id="name-{id}"
					name="name-{id}"
					type="text"
					placeholder="Living Room Lights"
					autocomplete="off"
					value={group?.name ?? ''}
					required
				/>
				<Field.Description>A descriptive name for this group.</Field.Description>
			</Field.Field>

			<Field.Field>
				<Field.Label for="type-{id}">Service Type</Field.Label>
				{#if isEditing}
					<Input id="type-{id}" disabled value={typeLabel(group?.type ?? Service_ServiceType.ON_OFF)} />
					<Field.Description>The service type cannot be changed after creation.</Field.Description>
				{:else}
					<select
						id="type-{id}"
						name="type-{id}"
						bind:value={selectedType}
						class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
					>
						{#each serviceTypeOptions as opt (opt.value)}
							<option value={opt.value}>{opt.label}</option>
						{/each}
					</select>
					<Field.Description>All group members must share this service type.</Field.Description>
				{/if}
			</Field.Field>
		</Field.Group>

		<Field.Group>
			<Field.Set>
				<Field.Label>Members</Field.Label>
				<Field.Description>Select devices to include in this group.</Field.Description>
				<div class="flex flex-col gap-0.5 max-h-52 overflow-y-auto rounded-md border bg-background p-1">
					{#if availableMembers.length === 0}
						<p class="text-muted-foreground text-sm py-3 text-center">
							No devices with service type <strong>{typeLabel(selectedType)}</strong> found.
						</p>
					{/if}
					{#each availableMembers as member (member.key)}
						<label
							class="flex items-center gap-2.5 rounded-sm px-2 py-1.5 hover:bg-accent cursor-pointer text-sm select-none"
						>
							<input
								type="checkbox"
								checked={selectedMembers.has(member.key)}
								onchange={() => toggleMember(member.key)}
								class="size-4 rounded accent-primary cursor-pointer"
							/>
							<span class="flex-1 font-medium">{member.label}</span>
						</label>
					{/each}
				</div>
				<Field.Description>
					{selectedMembers.size} member{selectedMembers.size === 1 ? '' : 's'} selected.
				</Field.Description>
			</Field.Set>
		</Field.Group>

		{#if error}
			<Field.Error>{error}</Field.Error>
		{/if}

		<Field.Field>
			<Button type="submit" class="cursor-pointer" disabled={submitting}>
				{isEditing ? 'Save Changes' : 'Create Group'}
			</Button>
		</Field.Field>
	</Field.Set>
</form>
