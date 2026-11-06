<script lang="ts">
	// The services of one endpoint. The root endpoint holds the device-level
	// services and has no title; any other endpoint is titled with its display
	// label, which an admin can rename in place.
	import type { DevicesStoreDevice } from '$lib/stores/devices-stream';
	import { Service_ServiceType, type Endpoint } from '$lib/api/v2/clients/client_service_pb';
	import { ServiceEnumerator } from '$lib/components/wh/service';
	import { endpointLabel, isRootEndpoint } from '$lib/device';
	import { SetEndpointLabel } from '$lib/stores/requests';
	import { toSentenceCase } from '$lib/tools/headline-case';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input/index.js';
	import { cn } from '$lib/utils';
	import { CheckIcon, PencilIcon, XIcon } from '@lucide/svelte';

	let {
		device,
		endpoint,
		editable = false,
		compact = false
	}: {
		device: DevicesStoreDevice;
		endpoint: Endpoint;
		/** Show the rename control (admins only). */
		editable?: boolean;
		/** The device card layout: services in a scrolling row, a smaller title. */
		compact?: boolean;
	} = $props();

	const isRoot = $derived(isRootEndpoint(endpoint));
	const title = $derived(endpointLabel(endpoint));
	const deviceName = $derived(device.name ? device.name : device.id);

	// Info and online are shown in the device header, not as services.
	const services = $derived(
		endpoint.services.filter(
			(srv) => srv.typ !== Service_ServiceType.INFO && srv.typ !== Service_ServiceType.ONLINE
		)
	);

	let editing = $state(false);
	let draft = $state('');
	let saving = $state(false);
	let error = $state<string | null>(null);

	function startEdit() {
		draft = endpoint.userLabel;
		error = null;
		editing = true;
	}

	async function save(event: SubmitEvent) {
		event.preventDefault();
		saving = true;
		error = null;
		// An empty label clears the user's label, so the bridge's shows again.
		const err = await SetEndpointLabel(device.id, endpoint.id, draft.trim());
		saving = false;
		if (err) {
			error = err.rawMessage || err.message;
		} else {
			// The new label arrives with the devices stream.
			editing = false;
		}
	}
</script>

{#if services.length > 0 || (!isRoot && editable)}
	<div class="grid gap-2">
		{#if !isRoot}
			{#if editing}
				<form onsubmit={save} class="flex flex-wrap items-center gap-2">
					<Input
						type="text"
						class="max-w-64"
						placeholder={endpoint.label || `Endpoint ${endpoint.id}`}
						autocomplete="off"
						bind:value={draft}
						disabled={saving}
					/>
					<Button type="submit" size="icon-sm" class="cursor-pointer" aria-label="Save label" disabled={saving}>
						<CheckIcon />
					</Button>
					<Button
						type="button"
						variant="secondary"
						size="icon-sm"
						class="cursor-pointer"
						aria-label="Cancel"
						disabled={saving}
						onclick={() => (editing = false)}
					>
						<XIcon />
					</Button>
					<span class="text-xs text-muted-foreground">Leave empty to use the default label.</span>
					{#if error}
						<span class="w-full text-sm text-destructive">{toSentenceCase(error)}</span>
					{/if}
				</form>
			{:else}
				<div class="flex items-center gap-2">
					<h3 class={cn('font-medium', compact ? 'text-sm text-muted-foreground' : 'text-sm')}>{title}</h3>
					{#if editable}
						<Button
							variant="ghost"
							size="icon-sm"
							class="size-6 cursor-pointer"
							aria-label="Rename endpoint"
							onclick={startEdit}
						>
							<PencilIcon class="size-3.5" />
						</Button>
					{/if}
				</div>
			{/if}
		{/if}

		{#if services.length > 0}
			<div
				class={cn(
					'flex gap-2',
					compact ? 'flex-row overflow-x-scroll' : 'flex-col md:flex-row overflow-x-auto'
				)}
			>
				{#each services as srv (srv.id)}
					<ServiceEnumerator
						showDeviceName={false}
						naturalWidth={compact}
						{deviceName}
						deviceID={device.id}
						endpointID={endpoint.id}
						deviceType={device.typ}
						online={device.online}
						service={srv}
					/>
				{/each}
			</div>
		{/if}
	</div>
{/if}
