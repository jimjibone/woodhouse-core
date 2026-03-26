<script lang="ts">
	import { onDestroy } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { DevicesStore, type DevicesStoreDevice } from '$lib/stores/devices-stream';
	import { ServiceEnumerator } from '$lib/components/wh/service';
	import { ServiceSchema } from '$lib/api/v1/clients/client_service_pb';
	import { toJsonString } from '@bufbuild/protobuf';
	import { attributeToDate } from '$lib/tools/time';
	import { useConnectionContext } from '$lib/stores/connection-status.svelte';
	import { SendRemoveDeviceRequest } from '$lib/stores/requests';
	import Button from '$lib/components/ui/button/button.svelte';
	import Dialog from '$lib/components/wh/ui/dialog.svelte';
	import * as Field from '$lib/components/ui/field/index.js';
	import * as Collapsible from '$lib/components/ui/collapsible/index.js';
	import TimeSince from '$lib/components/wh/ui/time-since.svelte';
	import { type ConnectError } from '@connectrpc/connect';
	import { toSentenceCase } from '$lib/tools/headline-case';
	import { cn } from '$lib/utils';
	import {
		Trash2Icon,
		ChevronDownIcon,
		BatteryWarningIcon,
		BatteryLowIcon,
		BatteryMediumIcon,
		BatteryFullIcon
	} from '@lucide/svelte';

	const deviceID = page.params.slug;
	const connStatus = useConnectionContext();

	let connected = $state(false);
	let backoff = $state(0);
	let dev = $state<DevicesStoreDevice | undefined>(undefined);

	onDestroy(
		DevicesStore.subscribe((store) => {
			connected = store.connected;
			backoff = store.backoff;
			connStatus.set(store.connected, !store.connected && store.backoff > 0);
			for (const it of store.devices) {
				if (it.id === deviceID) {
					dev = it;
					break;
				}
			}
		})
	);
	onDestroy(() => connStatus.reset());

	let removeConfirmOpen = $state(false);
	let removeError = $state<ConnectError | null>(null);
	let removing = $state(false);
	let rawOpen = $state(false);

	async function handleRemove() {
		removing = true;
		removeError = null;
		const err = await SendRemoveDeviceRequest(deviceID!);
		removing = false;
		if (err) {
			removeError = err;
		} else {
			removeConfirmOpen = false;
			goto('/devices');
		}
	}
</script>

<main class="grid gap-6">
	{#if dev}
		{@const deviceName = dev.name ? dev.name : dev.id}

		<!-- Header -->
		<div class="flex items-start justify-between gap-4">
			<div class="grid gap-1.5 min-w-0">
				<h1 class={cn('text-2xl font-bold leading-tight truncate', !dev.name && 'font-mono')}>
					{deviceName}
				</h1>
				<div class="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm">
					<span class="flex items-center gap-1.5">
						<span class={cn('size-2 rounded-full shrink-0', dev.online ? 'bg-green-500' : 'bg-muted-foreground/40')}
						></span>
						<span class={dev.online ? 'text-green-600' : 'text-muted-foreground'}>
							{dev.online ? 'Online' : 'Offline'}
						</span>
					</span>
					{#if dev.batteryLevel}
						<span class="text-muted-foreground/40">·</span>
						<span
							class={cn(
								'flex items-center gap-1',
								dev.batteryLevel < 20
									? 'text-red-500'
									: dev.batteryLevel < 33
										? 'text-orange-500'
										: 'text-muted-foreground'
							)}
						>
							{#if dev.batteryLevel < 20}
								<BatteryWarningIcon class="size-4" />
							{:else if dev.batteryLevel < 33}
								<BatteryLowIcon class="size-4" />
							{:else if dev.batteryLevel < 66}
								<BatteryMediumIcon class="size-4" />
							{:else}
								<BatteryFullIcon class="size-4" />
							{/if}
							{Number(dev.batteryLevel)}%
						</span>
					{/if}
					{#if dev.lastSeen}
						<span class="text-muted-foreground/40">·</span>
						<span class="text-muted-foreground">Last seen</span>
						<TimeSince past={attributeToDate(dev.lastSeen)} />
					{/if}
				</div>
			</div>
			<Button
				variant="destructive"
				size="icon"
				class="size-8 shrink-0 cursor-pointer"
				onclick={() => {
					removeError = null;
					removeConfirmOpen = true;
				}}
			>
				<Trash2Icon />
			</Button>
		</div>

		<!-- Device Info Card -->
		<div class="rounded-xl border bg-card/50 p-4 shadow-sm">
			<div class="grid gap-2.5">
				{#if dev.name}
					<div class="grid grid-cols-[9rem_1fr] items-baseline gap-2">
						<span class="text-sm text-muted-foreground shrink-0">Device Name</span>
						<span class="text-sm">{dev.name}</span>
					</div>
				{/if}
				<div class="grid grid-cols-[9rem_1fr] items-baseline gap-2">
					<span class="text-sm text-muted-foreground shrink-0">Device ID</span>
					<span class="font-mono text-sm break-all">{dev.id}</span>
				</div>
				<div class="grid grid-cols-[9rem_1fr] items-baseline gap-2">
					<span class="text-sm text-muted-foreground shrink-0">Client ID</span>
					<span class="font-mono text-sm break-all">{dev.clientID || 'Unknown'}</span>
				</div>
			</div>
		</div>

		<!-- Services -->
		{#if dev.services.length > 0}
			<div class="grid gap-3">
				<h2 class="text-sm font-medium text-muted-foreground">Services</h2>
				<div class="flex flex-col md:flex-row gap-2 overflow-x-auto">
					{#each dev.services as srv (srv.id)}
						<ServiceEnumerator
							showDeviceName={false}
							{deviceName}
							deviceID={dev.id}
							online={dev.online}
							service={srv}
						/>
					{/each}
				</div>
			</div>
		{/if}

		<!-- Raw JSON (collapsible) -->
		<Collapsible.Root bind:open={rawOpen}>
			<Collapsible.Trigger
				class="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors cursor-pointer select-none"
			>
				<ChevronDownIcon class={cn('size-4 transition-transform duration-200', rawOpen && 'rotate-180')} />
				Raw service data
			</Collapsible.Trigger>
			<Collapsible.Content class="grid gap-2 pt-3">
				{#each dev.services as srv (srv.id)}
					<div class="min-w-0 overflow-x-auto font-mono bg-muted px-4 py-3 rounded-lg whitespace-pre text-xs">
						{toJsonString(ServiceSchema, srv, { prettySpaces: 2 })}
					</div>
				{/each}
			</Collapsible.Content>
		</Collapsible.Root>

		<!-- Remove Device Dialog -->
		<Dialog bind:open={removeConfirmOpen} title="Remove Device">
			<div class="flex flex-col gap-4 pt-2">
				<p class="text-sm text-muted-foreground">
					Are you sure you want to remove <strong class="text-foreground">{deviceName}</strong>? This action cannot be
					undone.
				</p>

				{#if removeError}
					<Field.Error>{toSentenceCase(removeError.rawMessage)}</Field.Error>
				{/if}

				<div class="flex gap-2 justify-end">
					<Button
						variant="secondary"
						class="cursor-pointer"
						onclick={() => (removeConfirmOpen = false)}
						disabled={removing}
					>
						Cancel
					</Button>
					<Button variant="destructive" class="cursor-pointer" onclick={handleRemove} disabled={removing}>
						{removing ? 'Removing…' : 'Remove Device'}
					</Button>
				</div>
			</div>
		</Dialog>
	{:else}
		<!-- Device Not Found -->
		<div class="rounded-xl border bg-card/50 p-4 shadow-sm grid gap-3">
			<h2 class="font-semibold">Device not found</h2>
			<div class="grid gap-2">
				<div class="grid grid-cols-[9rem_1fr] items-baseline gap-2">
					<span class="text-sm text-muted-foreground shrink-0">Device ID</span>
					<span class="font-mono text-sm break-all">{deviceID}</span>
				</div>
				<div class="grid grid-cols-[9rem_1fr] items-baseline gap-2">
					<span class="text-sm text-muted-foreground shrink-0">Server</span>
					<span class="font-mono text-sm">
						{connected ? 'Connected' : 'Disconnected (backoff=' + backoff + 'ms)'}
					</span>
				</div>
			</div>
		</div>
	{/if}
</main>
