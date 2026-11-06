<script lang="ts">
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import {
		MusicIcon,
		PauseIcon,
		PlayIcon,
		SkipBackIcon,
		SkipForwardIcon,
		SquareIcon
	} from '@lucide/svelte';
	import { Button } from '$lib/components/ui/button';
	import { OthersContent } from '$lib/components/wh/attributes';
	import { attrDuration, attrEnum, attrText, findAttr, findCommand, formatValue } from '$lib/device';
	import { toHeadlineCase } from '$lib/tools/headline-case';
	import { ImagesStore, imageKey, registerSizeHint, unregisterSizeHint } from '$lib/stores/images-stream';
	import { MediaPlayback } from '$lib/schema/zz_services';

	let { deviceID, endpointID, service, ...rest }: StandardProps = $props();

	const playState = $derived(attrEnum(service, MediaPlayback.attributes.state));
	const nowPlaying = $derived(attrText(service, MediaPlayback.attributes.nowPlaying));
	const title = $derived(attrText(service, MediaPlayback.attributes.title));
	const artist = $derived(attrText(service, MediaPlayback.attributes.artist));
	const album = $derived(attrText(service, MediaPlayback.attributes.album));
	const elapsedAttr = $derived(findAttr(service, MediaPlayback.attributes.elapsed));
	const durationAttr = $derived(findAttr(service, MediaPlayback.attributes.duration));
	const elapsed = $derived(attrDuration(service, MediaPlayback.attributes.elapsed));
	const duration = $derived(attrDuration(service, MediaPlayback.attributes.duration));
	const playing = $derived(playState === 'playing');

	// One line for the card: the bridge's summary, else title and artist.
	const summary = $derived(
		nowPlaying || [title, artist].filter((s) => s !== undefined && s !== '').join(' - ')
	);

	const has = (id: string) => findCommand(service, id) !== undefined;

	// svelte-ignore state_referenced_locally
	const serviceAction = new ServiceAction(deviceID, endpointID, service.id);

	// Cover art comes from the shared image store. The server needs a size
	// hint for the image, which is registered while the popup is open.
	const artID = MediaPlayback.attributes.art;
	const hasArt = $derived(findAttr(service, artID) !== undefined);
	const art = $derived($ImagesStore.images.get(imageKey(deviceID, endpointID, service.id, artID)));
	let drawerOpen = $state(false);
	const hintID = Symbol();
	$effect(() => {
		if (!drawerOpen || !hasArt) return;
		const [dev, ep, srv] = [deviceID, endpointID, service.id];
		registerSizeHint(dev, ep, srv, artID, hintID, 512, 512);
		return () => unregisterSizeHint(dev, ep, srv, artID, hintID);
	});
</script>

{#snippet icon()}
	{#if playing}
		<PlayIcon />
	{:else}
		<MusicIcon />
	{/if}
{/snippet}

{#snippet details()}
	{#if playState !== undefined}
		<p>{toHeadlineCase(playState)}</p>
	{/if}
	{#if summary !== ''}
		<p class="text-muted-foreground">{summary}</p>
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
	iconclass={playing ? 'bg-green-400 text-black' : false}
	{details}
	bind:drawerOpen
>
	{#if art}
		<img src={art.url} alt="Cover art" class="mb-4 max-h-64 mx-auto rounded-md" />
	{/if}
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if playState !== undefined}
			<div>State</div>
			<div class="col-span-2">{toHeadlineCase(playState)}</div>
		{/if}
		{#if title}
			<div>Title</div>
			<div class="col-span-2">{title}</div>
		{/if}
		{#if artist}
			<div>Artist</div>
			<div class="col-span-2">{artist}</div>
		{/if}
		{#if album}
			<div>Album</div>
			<div class="col-span-2">{album}</div>
		{/if}
		{#if elapsedAttr !== undefined && elapsed !== undefined}
			<div>Position</div>
			<div class="col-span-2">
				{formatValue(elapsedAttr.value, elapsedAttr.spec)}
				{#if durationAttr !== undefined && duration !== undefined}
					/ {formatValue(durationAttr.value, durationAttr.spec)}
				{/if}
			</div>
		{/if}
	</div>

	<div class="flex flex-row justify-center gap-2 pt-4">
		{#if has(MediaPlayback.commands.previous)}
			<Button
				variant="secondary"
				size="icon"
				class="cursor-pointer"
				aria-label="Previous"
				onclick={() => serviceAction.command(MediaPlayback.commands.previous)}
			>
				<SkipBackIcon />
			</Button>
		{/if}
		{#if playing && has(MediaPlayback.commands.pause)}
			<Button
				size="icon"
				class="cursor-pointer"
				aria-label="Pause"
				onclick={() => serviceAction.command(MediaPlayback.commands.pause)}
			>
				<PauseIcon />
			</Button>
		{:else if has(MediaPlayback.commands.play)}
			<Button
				size="icon"
				class="cursor-pointer"
				aria-label="Play"
				onclick={() => serviceAction.command(MediaPlayback.commands.play)}
			>
				<PlayIcon />
			</Button>
		{/if}
		{#if has(MediaPlayback.commands.stop)}
			<Button
				variant="secondary"
				size="icon"
				class="cursor-pointer"
				aria-label="Stop"
				onclick={() => serviceAction.command(MediaPlayback.commands.stop)}
			>
				<SquareIcon />
			</Button>
		{/if}
		{#if has(MediaPlayback.commands.next)}
			<Button
				variant="secondary"
				size="icon"
				class="cursor-pointer"
				aria-label="Next"
				onclick={() => serviceAction.command(MediaPlayback.commands.next)}
			>
				<SkipForwardIcon />
			</Button>
		{/if}
	</div>

	<OthersContent
		{deviceID}
		{endpointID}
		{service}
		action={serviceAction}
		exclude={[
			MediaPlayback.attributes.state,
			MediaPlayback.attributes.nowPlaying,
			MediaPlayback.attributes.title,
			MediaPlayback.attributes.artist,
			MediaPlayback.attributes.album,
			MediaPlayback.attributes.elapsed,
			MediaPlayback.attributes.duration,
			MediaPlayback.attributes.art
		]}
		excludeCommands={[
			MediaPlayback.commands.play,
			MediaPlayback.commands.pause,
			MediaPlayback.commands.stop,
			MediaPlayback.commands.next,
			MediaPlayback.commands.previous
		]}
	/>
</ServiceRoot>
