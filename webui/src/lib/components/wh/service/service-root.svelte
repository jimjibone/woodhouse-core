<script lang="ts">
	import { cn } from '$lib/utils';
	import { ServiceSchema, type Service, type TimeValue } from '$lib/api/v1/clients/client_service_pb';
	import { type Snippet } from 'svelte';
	import {
		HeartIcon,
		HeartOffIcon,
		EllipsisIcon,
		UnplugIcon,
		BugIcon,
		Loader2Icon,
		SquareDashedIcon,
		LampIcon,
		BatteryWarningIcon,
		BatteryLowIcon,
		BatteryMediumIcon,
		BatteryFullIcon
	} from '@lucide/svelte';
	import { IsMobile } from '$lib/hooks/is-mobile.svelte.js';
	import * as Dialog from '$lib/components/ui/dialog';
	import * as Drawer from '$lib/components/ui/drawer';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { Button } from '$lib/components/ui/button';
	import { SendFavoriteRequest } from '$lib/stores/requests';
	import { TooltipIcon } from '$lib/components/wh/buttons';
	import { toJsonString } from '@bufbuild/protobuf';
	import { slide } from 'svelte/transition';
	import { toHeadlineCase } from '$lib/tools/headline-case';
	import TimeSince from '$lib/components/wh/ui/time-since.svelte';

	const isMobile = new IsMobile();

	export type StandardProps = {
		deviceName: string;
		showDeviceName?: boolean;
		deviceID: string;
		online: boolean;
		lastSeen?: Date;
		batteryLevel?: bigint;
		service: Service;
		naturalWidth?: boolean;
	};

	export type Props = StandardProps & {
		actionPending?: boolean;
		/**
		 * A signal from the parent indicating that an error occurred.
		 * Used to temporarily flash the component red.
		 *
		 * Recommended format: a timestamp or counter that changes on each error - e.g. `Date.now()`
		 */
		errorSignal?: number | null;
		icon?: Snippet;
		iconclass?: string | boolean;
		iconstyle?: string;
		oniconclick?: () => void;
		details?: Snippet;
		children?: Snippet<[]>;
		drawerOpen?: boolean;
	};

	let {
		deviceName,
		showDeviceName = true,
		deviceID,
		online,
		lastSeen = undefined,
		batteryLevel = undefined,
		service,
		actionPending = false,
		errorSignal = null,
		icon = undefined,
		iconclass = false,
		iconstyle = '',
		oniconclick,
		details = undefined,
		children = undefined,
		drawerOpen = $bindable(false),
		naturalWidth = false
	}: Props = $props();

	let serviceTitle = $derived.by(() => {
		if (showDeviceName) {
			let dev = deviceName !== '' ? deviceName : deviceID;
			let srv = service.alias ? ': ' + toHeadlineCase(service.alias) : '';
			return dev + srv;
		}
		return service.alias ? service.alias : '';
	});

	let popupTitle = $derived.by(() => {
		let dev = deviceName !== '' ? deviceName : deviceID;
		let srv = ': ' + toHeadlineCase(service.alias ? service.alias : service.id);
		return dev + srv;
	});

	let rawPanelOpen = $state(false);

	let toggleFavorite = async () => {
		SendFavoriteRequest(deviceID, service.id, !service.favorite);
	};

	let isError = $state(false);
	$effect(() => {
		if (errorSignal !== null) {
			// Trigger reflow to restart animation
			isError = false;
			requestAnimationFrame(() => {
				isError = true;
				setTimeout(() => (isError = false), 500);
			});
		}
	});
</script>

<button
	class={cn(
		'w-full max-w-full rounded-xl border bg-card/50 hover:bg-card/70 p-2 text-card-foreground shadow-sm text-left cursor-pointer text-base md:text-sm',
		!online && 'bg-muted/80 hover:bg-muted/90',
		isError && 'shake'
	)}
	onclick={(event) => {
		event.stopPropagation();
		drawerOpen = true;
	}}
>
	<div class="flex flex-row">
		<div class="shrink">
			<div class="grid h-full place-content-center">
				{#if oniconclick !== undefined}
					<span
						class={cn(
							'p-3 md:p-2 rounded-full bg-secondary text-secondary-foreground transition-[background-color,color] duration-200 ease-linear cursor-pointer',
							iconclass
						)}
						style={iconstyle}
						onclick={(event) => {
							event.stopPropagation();
							oniconclick();
						}}
						role="button"
						tabindex="0"
						onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && oniconclick()}
					>
						{#if actionPending}
							<Loader2Icon class="animate-spin" />
						{:else if icon}
							{@render icon()}
						{:else}
							<SquareDashedIcon />
						{/if}
					</span>
				{:else}
					<span
						class={cn('p-3 md:p-2 rounded-full bg-secondary text-secondary-foreground', iconclass)}
						style={iconstyle}
					>
						{#if icon}
							{@render icon()}
						{:else}
							<SquareDashedIcon />
						{/if}
					</span>
				{/if}
			</div>
		</div>
		<div class="grow">
			<div class="h-full max-w-full flex flex-col justify-center gap-0">
				{#if serviceTitle !== ''}
					<div class="pl-2 pr-1 grid grid-cols-[1fr_auto] gap-1 max-w-full">
						<span class="font-semibold whitespace-pre overflow-x-auto">
							{serviceTitle}
						</span>
						<span class="flex flex-row gap-2 text-sm items-center whitespace-pre">
							{#if lastSeen}
								<TimeSince past={lastSeen} class="text-sm" />
							{/if}
						</span>
					</div>
					<!-- </div> -->
				{/if}
				{#if details}
					<!-- <div class="pl-2 pr-1 flex h-full flex-col justify-center gap-0"> -->
					<div class="pl-2 pr-1 grid grid-cols-[1fr_auto] gap-1 max-w-full">
						<div class="flex flex-row gap-2 whitespace-pre overflow-x-auto">
							{@render details()}
						</div>
						{#if batteryLevel}
							<!-- <span class={cn("shrink flex flex-row gap-1 text-sm items-center text-muted-foreground", batteryLevel < 33 && "text-warning-foreground", batteryLevel < 20 && "text-error-foreground")}> -->
							<span
								class={cn(
									'flex flex-row gap-0 text-sm text-muted-foreground',
									batteryLevel < 33 && 'text-warning-foreground',
									batteryLevel < 20 && 'text-error-foreground'
								)}
							>
								{#if batteryLevel < 20}
									<BatteryWarningIcon class="size-5" />
								{:else if batteryLevel < 33}
									<BatteryLowIcon class="size-5" />
								{:else if batteryLevel < 66}
									<BatteryMediumIcon class="size-5" />
								{:else}
									<BatteryFullIcon class="size-5" />
								{/if}
								{Number(batteryLevel)}%
							</span>
						{/if}
					</div>
				{/if}
			</div>
		</div>
	</div>
	{#if isMobile.current}
		<Drawer.Root bind:open={drawerOpen}>
			<Drawer.Content class="min-h-[50%]">
				<div class={cn('w-full mx-auto flex flex-col overflow-auto p-4 rounded-t-[10px] pt-0', isError && 'shake')}>
					<Drawer.Header>
						<Drawer.Title class="flex flex-row gap-2 items-center">
							<span class="grow text-lg">
								{popupTitle}
							</span>
							{#if service.favorite}
								<TooltipIcon variant="default" tooltip="Favorite">
									<HeartIcon />
								</TooltipIcon>
							{/if}
							{#if !online}
								<TooltipIcon variant="destructive" tooltip="Offline">
									<UnplugIcon />
								</TooltipIcon>
							{/if}
							{@render drawerMenu()}
						</Drawer.Title>
					</Drawer.Header>

					<div class="px-4 pb-4 flex flex-col">
						{#if actionPending}
							<span transition:slide={{ duration: 200 }} class="p-2 mb-4 bg-warning rounded-md flex gap-1">
								<Loader2Icon class="animate-spin" />
								Action pending...
							</span>
						{/if}

						{#if children}
							{@render children()}
						{:else}
							{@render rawContent()}
						{/if}
					</div>
				</div>
			</Drawer.Content>
		</Drawer.Root>
	{:else}
		<Dialog.Root bind:open={drawerOpen}>
			<Dialog.Content class="max-h-[90%] overflow-y-auto" showCloseButton={false}>
				<div class={cn('grid gap-1', isError && 'shake')}>
					<Dialog.Header>
						<Dialog.Title class="flex flex-row gap-2 items-center">
							<span class="grow">
								{popupTitle}
							</span>
							{#if service.favorite}
								<TooltipIcon variant="default" tooltip="Favorite">
									<HeartIcon />
								</TooltipIcon>
							{/if}
							{#if !online}
								<TooltipIcon variant="destructive" tooltip="Offline">
									<UnplugIcon />
								</TooltipIcon>
							{/if}
							{@render drawerMenu()}
						</Dialog.Title>
					</Dialog.Header>

					{#if actionPending}
						<span transition:slide={{ duration: 200 }} class="p-2 mt-4 bg-warning rounded-md flex gap-1">
							<Loader2Icon class="animate-spin" />
							Action pending...
						</span>
					{/if}

					<div class="pt-4"></div>
					{#if children}
						{@render children()}
					{:else}
						{@render rawContent()}
					{/if}
				</div>
			</Dialog.Content>
		</Dialog.Root>
	{/if}
	<Dialog.Root bind:open={rawPanelOpen}>
		<Dialog.Content class="max-h-[90%] overflow-y-auto">
			<div class={cn('grid gap-1', isError && 'shake')}>
				<Dialog.Header>
					<Dialog.Title class="pb-3">Raw Service</Dialog.Title>
				</Dialog.Header>

				{@render rawContent()}
			</div>
		</Dialog.Content>
	</Dialog.Root>
</button>

{#snippet drawerMenu()}
	<DropdownMenu.Root>
		<DropdownMenu.Trigger>
			{#snippet child({ props })}
				<Button {...props} class="cursor-pointer" variant="outline" size="icon"><EllipsisIcon /></Button>
			{/snippet}
		</DropdownMenu.Trigger>
		<DropdownMenu.Content class="w-56" align="start">
			<DropdownMenu.Item onclick={toggleFavorite}>
				{#if service.favorite}
					<HeartOffIcon /> Unset Favorite
				{:else}
					<HeartIcon /> Set favorite
				{/if}
			</DropdownMenu.Item>
			<DropdownMenu.Item
				onclick={() => {
					rawPanelOpen = true;
				}}
			>
				<BugIcon /> Raw View
			</DropdownMenu.Item>
			<DropdownMenu.Item>
				{#snippet child({ props })}
					<a {...props} onclick={(event) => event.stopPropagation()} href={'/devices/' + deviceID}>
						<LampIcon /> Go to Device
					</a>
				{/snippet}
			</DropdownMenu.Item>
		</DropdownMenu.Content>
	</DropdownMenu.Root>
{/snippet}

{#snippet rawContent()}
	<div class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 items-center">
		<div>Device Name</div>
		<div class="font-mono bg-muted p-1 rounded-md">{deviceName}</div>
		<div>Device ID</div>
		<div class="font-mono bg-muted p-1 rounded-md">{deviceID}</div>
		<div>Online</div>
		<div class="font-mono bg-muted p-1 rounded-md">{online}</div>
		{#if lastSeen}
			<div>Last Seen</div>
			<div class="font-mono bg-muted p-1 rounded-md">{lastSeen.toLocaleString()}</div>
		{/if}
		{#if batteryLevel}
			<div>Battery</div>
			<div class="font-mono bg-muted p-1 rounded-md">{Number(batteryLevel)}%</div>
		{/if}
		<div class="col-span-2">Service:</div>
	</div>
	<div class="min-w-0 overflow-x-scroll font-mono bg-muted px-4 py-2 rounded-md whitespace-pre text-sm">
		{toJsonString(ServiceSchema, service, { prettySpaces: 2 })}
	</div>
{/snippet}
