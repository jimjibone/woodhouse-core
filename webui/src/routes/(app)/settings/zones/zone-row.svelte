<script lang="ts">
	import type { Zone } from '$lib/api/v1/clients/zone_pb';
	import Button from '$lib/components/ui/button/button.svelte';
	import { PencilIcon, Trash2Icon, LampIcon } from '@lucide/svelte';
	import Dialog from '$lib/components/wh/ui/dialog.svelte';
	import * as Field from '$lib/components/ui/field/index.js';
	import { RemoveZone } from '@/stores/requests';
	import type { ConnectError } from '@connectrpc/connect';
	import { toSentenceCase } from '@/tools/headline-case';
	import { zoneIcon } from '$lib/zone-icons';
	import ZoneForm from './zone-form.svelte';

	let { zone }: { zone: Zone } = $props();

	let editOpen = $state(false);
	let deleteConfirmOpen = $state(false);
	let deleteError: ConnectError | null = $state(null);
	let deleting = $state(false);

	const ZoneIcon = $derived(zoneIcon(zone.icon));

	async function handleDelete() {
		deleting = true;
		deleteError = null;
		const err = await RemoveZone(zone.id);
		deleting = false;
		if (err) {
			deleteError = err;
		} else {
			deleteConfirmOpen = false;
		}
	}
</script>

<div class="rounded-lg border bg-card/50 p-2 text-card-foreground shadow-sm flex flex-row gap-2 items-center">
	<div class="shrink pl-2 flex flex-row gap-2.5 items-center min-w-0">
		<ZoneIcon class="size-5 text-muted-foreground shrink-0" />
		<span class="font-semibold truncate">{zone.name}</span>
	</div>
	<div class="grow flex flex-row items-center justify-center gap-1.5">
		<LampIcon class="size-4 text-muted-foreground shrink-0" />
		<span class="text-muted-foreground text-sm">
			{zone.deviceIds.length}
			{zone.deviceIds.length === 1 ? 'device' : 'devices'}
		</span>
	</div>
	<div class="shrink-0 flex flex-row pr-2 gap-2 items-center">
		<Button variant="secondary" size="icon" class="size-8 cursor-pointer" onclick={() => (editOpen = true)}>
			<PencilIcon />
		</Button>
		<Button
			variant="destructive"
			size="icon"
			class="size-8 cursor-pointer"
			onclick={() => {
				deleteError = null;
				deleteConfirmOpen = true;
			}}
		>
			<Trash2Icon />
		</Button>
	</div>
</div>

<Dialog bind:open={editOpen}>
	<ZoneForm {zone} onSuccess={() => (editOpen = false)} />
</Dialog>

<Dialog bind:open={deleteConfirmOpen} title="Delete Zone">
	<div class="flex flex-col gap-4 pt-2">
		<p class="text-sm text-muted-foreground">
			Are you sure you want to delete <strong class="text-foreground">{zone.name}</strong>? Its
			{zone.deviceIds.length}
			{zone.deviceIds.length === 1 ? 'device' : 'devices'} will be left unassigned, not removed.
		</p>

		{#if deleteError}
			<Field.Error>{toSentenceCase(deleteError.rawMessage)}</Field.Error>
		{/if}

		<div class="flex gap-2 justify-end">
			<Button variant="secondary" class="cursor-pointer" onclick={() => (deleteConfirmOpen = false)} disabled={deleting}>
				Cancel
			</Button>
			<Button variant="destructive" class="cursor-pointer" onclick={handleDelete} disabled={deleting}>
				{deleting ? 'Deleting…' : 'Delete'}
			</Button>
		</div>
	</div>
</Dialog>
