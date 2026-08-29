<script lang="ts">
	import type { Zone } from '$lib/api/v1/clients/zone_pb';
	import { ZonesStore as store } from '$lib/stores/zones-stream';
	import Button from '$lib/components/ui/button/button.svelte';
	import { onDestroy } from 'svelte';
	import { useConnectionContext } from '$lib/stores/connection-status.svelte';
	import { MapPinIcon, TriangleAlertIcon } from '@lucide/svelte';
	import ZoneRow from './zone-row.svelte';
	import Dialog from '$lib/components/wh/ui/dialog.svelte';
	import ZoneForm from './zone-form.svelte';
	import { toSentenceCaseLocalized } from '@/tools/headline-case';

	let dialogOpen = $state(false);
	let zones = $state<Zone[]>([]);
	let streamError = $state<string | null>(null);

	const connStatus = useConnectionContext();
	onDestroy(
		store.subscribe((update) => {
			zones = update.zones;
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
				<p class="font-medium">Unable to load zones</p>
				<p class="mt-0.5 text-destructive/80">{toSentenceCaseLocalized(streamError)}</p>
			</div>
		</div>
	{/if}
	<div class="pb-4">
		<Button class="cursor-pointer" onclick={() => (dialogOpen = true)}>
			<MapPinIcon />
			Add Zone
		</Button>
	</div>
	<div class="flex flex-col gap-4">
		{#each zones as zone (zone.id)}
			<ZoneRow {zone} />
		{:else}
			<p class="text-muted-foreground text-sm">
				No zones yet. A zone is a room or area — create one and it appears in the navigation for every user.
			</p>
		{/each}
	</div>
</main>

<Dialog bind:open={dialogOpen}>
	<ZoneForm onSuccess={() => (dialogOpen = false)} />
</Dialog>
