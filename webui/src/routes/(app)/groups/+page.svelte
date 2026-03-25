<script lang="ts">
	import type { Group } from '$lib/api/v1/clients/group_pb';
	import { GroupsStore as store } from '$lib/stores/groups-stream';
	import Button from '$lib/components/ui/button/button.svelte';
	import { onDestroy } from 'svelte';
	import { useConnectionContext } from '$lib/stores/connection-status.svelte';
	import { LayersIcon, TriangleAlertIcon } from '@lucide/svelte';
	import GroupRow from './group-row.svelte';
	import Dialog from '$lib/components/wh/ui/dialog.svelte';
	import GroupForm from './group-form.svelte';
	import { toSentenceCaseLocalized } from '@/tools/headline-case';

	let dialogOpen = $state(false);
	let groups = $state<Group[]>([]);
	let streamError = $state<string | null>(null);

	const connStatus = useConnectionContext();
	onDestroy(
		store.subscribe((update) => {
			groups = update.groups;
			streamError = update.error;
			connStatus.set(update.connected, !update.connected && update.backoff > 0);
		})
	);
	onDestroy(() => connStatus.reset());
</script>

<main>
	{#if streamError}
		<div
			class="mb-4 flex items-start gap-3 rounded-md border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive"
		>
			<TriangleAlertIcon class="mt-0.5 size-4 shrink-0" />
			<div>
				<p class="font-medium">Unable to load groups</p>
				<p class="mt-0.5 text-destructive/80">{toSentenceCaseLocalized(streamError)}</p>
			</div>
		</div>
	{/if}
	<div class="pb-4">
		<Button class="cursor-pointer" onclick={() => (dialogOpen = true)}>
			<LayersIcon />
			Add Group
		</Button>
	</div>
	<div class="flex flex-col gap-4">
		{#each groups as group (group.id)}
			<GroupRow {group} />
		{:else}
			<p class="text-muted-foreground text-sm">No groups yet.</p>
		{/each}
	</div>
</main>

<Dialog bind:open={dialogOpen}>
	<GroupForm onSuccess={() => (dialogOpen = false)} />
</Dialog>
