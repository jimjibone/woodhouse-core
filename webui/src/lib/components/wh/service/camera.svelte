<script lang="ts">
	import { ImageResponse_ImageStatus } from '$lib/api/v1/clients/client_service_pb';
	import { type StandardProps } from './service-root.svelte';
	import { SendImageRequest, SendFavoriteRequest } from '$lib/stores/requests';
	import {
		CameraIcon,
		RefreshCwIcon,
		Loader2Icon,
		EllipsisIcon,
		HeartIcon,
		HeartOffIcon,
		UnplugIcon,
		BugIcon,
		LampIcon
	} from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import { onMount } from 'svelte';
	import { cn } from '$lib/utils';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { Button } from '$lib/components/ui/button';
	import { ServiceSchema } from '$lib/api/v1/clients/client_service_pb';
	import { toJsonString } from '@bufbuild/protobuf';
	import * as Dialog from '$lib/components/ui/dialog';
	import { toHeadlineCase } from '$lib/tools/headline-case';

	let { deviceID, deviceName, showDeviceName = true, service, online, ...rest }: StandardProps = $props();

	let imageAttrID = $state('image');
	$effect(() => {
		for (const attr of service.attrs) {
			if (attr.image !== undefined) {
				imageAttrID = attr.id;
				break;
			}
		}
	});

	let imageURL: string | null = $state(null);
	let fetchedAt: Date | null = $state(null);
	let pending = $state(false);
	let rawPanelOpen = $state(false);

	const fetchImage = async () => {
		if (pending) return;
		pending = true;
		await SendImageRequest(deviceID, service.id, imageAttrID, (response) => {
			if (response.status === ImageResponse_ImageStatus.COMPLETE && response.data.length > 0) {
				if (imageURL !== null) URL.revokeObjectURL(imageURL);
				const blob = new Blob([response.data], { type: 'image/jpeg' });
				imageURL = URL.createObjectURL(blob);
				fetchedAt = new Date();
			} else if (response.status >= ImageResponse_ImageStatus.TIMEOUT) {
				toast.error('Image request failed', { description: response.details });
			}
		});
		pending = false;
	};

	onMount(() => {
		if (online) fetchImage();
	});

	let serviceTitle = $derived.by(() => {
		if (showDeviceName) {
			let dev = deviceName !== '' ? deviceName : deviceID;
			let srv = service.alias ? ': ' + toHeadlineCase(service.alias) : '';
			return dev + srv;
		}
		return service.alias ? toHeadlineCase(service.alias) : '';
	});

	const toggleFavorite = () => SendFavoriteRequest(deviceID, service.id, !service.favorite);
</script>

<div
	class={cn(
		'relative w-full min-w-64 flex-none overflow-hidden rounded-xl border bg-card/50 text-card-foreground shadow-sm',
		!online && 'opacity-60'
	)}
>
	<!-- Image area -->
	<div class="relative aspect-video w-full bg-black flex items-center justify-center">
		{#if imageURL !== null}
			<img src={imageURL} alt="Camera feed" class="w-full h-full object-cover" />
		{:else if pending}
			<Loader2Icon class="animate-spin text-white/60 size-8" />
		{:else}
			<CameraIcon class="text-white/30 size-10" />
		{/if}

		<!-- Top gradient + title bar -->
		<div class="absolute inset-x-0 top-0 h-12 bg-gradient-to-b from-black/60 to-transparent pointer-events-none"></div>
		<div class="absolute top-0 inset-x-0 flex items-center justify-between px-2 pt-1.5">
			{#if serviceTitle !== ''}
				<span class="text-xs font-semibold text-white drop-shadow truncate max-w-[75%]">{serviceTitle}</span>
			{/if}
			<div class="ml-auto flex items-center gap-1">
				{#if !online}
					<UnplugIcon class="size-3.5 text-white/80" />
				{/if}
				{#if service.favorite}
					<HeartIcon class="size-3.5 text-white/80 fill-white/80" />
				{/if}
			</div>
		</div>

		<!-- Bottom gradient + refresh + timestamp -->
		<div
			class="absolute inset-x-0 bottom-0 h-10 bg-gradient-to-t from-black/60 to-transparent pointer-events-none"
		></div>
		<div class="absolute bottom-0 inset-x-0 flex items-center justify-between px-2 pb-1.5">
			<span class="text-xs text-white/70">
				{#if fetchedAt !== null}
					{fetchedAt.toLocaleTimeString()}
				{:else if !pending}
					No image
				{/if}
			</span>
			<button
				class="flex items-center justify-center rounded-full p-1 text-white/80 hover:text-white hover:bg-white/20 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
				onclick={(e) => {
					e.stopPropagation();
					fetchImage();
				}}
				disabled={pending || !online}
				title="Refresh"
			>
				{#if pending}
					<Loader2Icon class="animate-spin size-4" />
				{:else}
					<RefreshCwIcon class="size-4" />
				{/if}
			</button>
		</div>
	</div>

	<!-- Menu button (outside image, bottom-right of card) -->
	<div class="absolute top-1.5 right-1.5">
		<DropdownMenu.Root>
			<DropdownMenu.Trigger>
				{#snippet child({ props })}
					<Button
						{...props}
						size="icon"
						class="size-6 rounded-full bg-black/40 hover:bg-black/60 text-white border-0 cursor-pointer"
						onclick={(e: MouseEvent) => e.stopPropagation()}
					>
						<EllipsisIcon class="size-3.5" />
					</Button>
				{/snippet}
			</DropdownMenu.Trigger>
			<DropdownMenu.Content class="w-48" align="end">
				<DropdownMenu.Item onclick={toggleFavorite}>
					{#if service.favorite}
						<HeartOffIcon /> Unset Favorite
					{:else}
						<HeartIcon /> Set Favorite
					{/if}
				</DropdownMenu.Item>
				<DropdownMenu.Item onclick={() => (rawPanelOpen = true)}>
					<BugIcon /> Raw View
				</DropdownMenu.Item>
				<DropdownMenu.Item>
					{#snippet child({ props })}
						<a {...props} onclick={(e: MouseEvent) => e.stopPropagation()} href={'/devices/' + deviceID}>
							<LampIcon /> Go to Device
						</a>
					{/snippet}
				</DropdownMenu.Item>
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	</div>
</div>

<!-- Raw view dialog -->
<Dialog.Root bind:open={rawPanelOpen}>
	<Dialog.Content class="max-h-[90%] overflow-y-auto">
		<Dialog.Header>
			<Dialog.Title class="pb-3">Raw Service</Dialog.Title>
		</Dialog.Header>
		<div class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 items-center text-sm">
			<div>Device Name</div>
			<div class="font-mono bg-muted p-1 rounded-md">{deviceName}</div>
			<div>Device ID</div>
			<div class="font-mono bg-muted p-1 rounded-md">{deviceID}</div>
			<div>Online</div>
			<div class="font-mono bg-muted p-1 rounded-md">{online}</div>
			<div class="col-span-2">Service:</div>
		</div>
		<div class="min-w-0 overflow-x-scroll font-mono bg-muted px-4 py-2 rounded-md whitespace-pre text-sm">
			{toJsonString(ServiceSchema, service, { prettySpaces: 2 })}
		</div>
	</Dialog.Content>
</Dialog.Root>
