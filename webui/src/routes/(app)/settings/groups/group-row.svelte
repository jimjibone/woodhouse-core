<script lang="ts">
	import type { Group } from '$lib/api/v1/clients/group_pb';
	import { Service_ServiceType } from '$lib/api/v1/clients/client_service_pb';
	import Button from '$lib/components/ui/button/button.svelte';
	import Badge from '$lib/components/ui/badge/badge.svelte';
	import { PencilIcon, Trash2Icon, UsersIcon } from '@lucide/svelte';
	import Dialog from '$lib/components/wh/ui/dialog.svelte';
	import * as Field from '$lib/components/ui/field/index.js';
	import { RemoveGroup } from '@/stores/requests';
	import type { ConnectError } from '@connectrpc/connect';
	import { toSentenceCase } from '@/tools/headline-case';
	import GroupForm from './group-form.svelte';

	let { group }: { group: Group } = $props();

	let editOpen = $state(false);
	let deleteConfirmOpen = $state(false);
	let deleteError: ConnectError | null = $state(null);
	let deleting = $state(false);

	const serviceTypeLabels: Partial<Record<Service_ServiceType, string>> = {
		[Service_ServiceType.RELAY]: 'Relay',
		[Service_ServiceType.LIGHTBULB]: 'Lightbulb',
		[Service_ServiceType.CLIMATE]: 'Climate',
		[Service_ServiceType.COVER]: 'Cover',
		[Service_ServiceType.CONTACT]: 'Contact',
		[Service_ServiceType.MOTION]: 'Motion',
		[Service_ServiceType.PRESENCE]: 'Presence',
		[Service_ServiceType.ENVIRONMENT]: 'Environment',
		[Service_ServiceType.BUTTON]: 'Button',
		[Service_ServiceType.INPUT]: 'Input',
		[Service_ServiceType.GENERIC]: 'Generic',
		[Service_ServiceType.BOOL]: 'Bool',
		[Service_ServiceType.INT]: 'Int',
		[Service_ServiceType.FLOAT]: 'Float',
		[Service_ServiceType.COLOR]: 'Color',
		[Service_ServiceType.CAMERA]: 'Camera',
		[Service_ServiceType.ENUM]: 'Enum',
		[Service_ServiceType.TEXT]: 'Text',
		[Service_ServiceType.DURATION]: 'Duration',
		[Service_ServiceType.TIME]: 'Time',
	};

	function typeLabel(type: Service_ServiceType): string {
		return serviceTypeLabels[type] ?? 'Unknown';
	}

	async function handleDelete() {
		deleting = true;
		deleteError = null;
		const err = await RemoveGroup(group.id);
		deleting = false;
		if (err) {
			deleteError = err;
		} else {
			deleteConfirmOpen = false;
		}
	}
</script>

<div class="rounded-lg border bg-card/50 p-2 text-card-foreground shadow-sm flex flex-row gap-2 items-center">
	<div class="shrink pl-2 flex flex-col gap-1 min-w-0">
		<span class="font-semibold truncate">{group.name}</span>
		<div>
			<Badge variant="secondary">{typeLabel(group.type)}</Badge>
		</div>
	</div>
	<div class="grow flex flex-row items-center justify-center gap-1.5">
		<UsersIcon class="size-4 text-muted-foreground shrink-0" />
		<span class="text-muted-foreground text-sm">
			{group.members.length}
			{group.members.length === 1 ? 'member' : 'members'}
		</span>
	</div>
	<div class="shrink-0 flex flex-row pr-2 gap-2 items-center">
		<Button
			variant="secondary"
			size="icon"
			class="size-8 cursor-pointer"
			onclick={() => (editOpen = true)}
		>
			<PencilIcon />
		</Button>
		<Button
			variant="destructive"
			size="icon"
			class="size-8 cursor-pointer"
			onclick={() => { deleteError = null; deleteConfirmOpen = true; }}
		>
			<Trash2Icon />
		</Button>
	</div>
</div>

<Dialog bind:open={editOpen}>
	<GroupForm {group} onSuccess={() => (editOpen = false)} />
</Dialog>

<Dialog bind:open={deleteConfirmOpen} title="Delete Group">
	<div class="flex flex-col gap-4 pt-2">
		<p class="text-sm text-muted-foreground">
			Are you sure you want to delete <strong class="text-foreground">{group.name}</strong>?
			This action cannot be undone.
		</p>

		{#if deleteError}
			<Field.Error>{toSentenceCase(deleteError.rawMessage)}</Field.Error>
		{/if}

		<div class="flex gap-2 justify-end">
			<Button
				variant="secondary"
				class="cursor-pointer"
				onclick={() => (deleteConfirmOpen = false)}
				disabled={deleting}
			>
				Cancel
			</Button>
			<Button
				variant="destructive"
				class="cursor-pointer"
				onclick={handleDelete}
				disabled={deleting}
			>
				{deleting ? 'Deleting…' : 'Delete'}
			</Button>
		</div>
	</div>
</Dialog>
