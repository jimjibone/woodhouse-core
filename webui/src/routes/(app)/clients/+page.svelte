<script lang="ts">
	import { onDestroy } from 'svelte';
	import Button from '$lib/components/ui/button/button.svelte';
	import TimeSince from '$lib/components/wh/ui/time-since.svelte';
	import { ClientsStore } from '$lib/stores/clients-stream';
	import { PairingRequestsStore } from '$lib/stores/pairing-requests-stream';
	import { ApprovePairing, DenyPairing, UnpairClient, ForgetClient } from '$lib/stores/requests';
	import type { Client, PairingRequest } from '$lib/api/v1/clients/client_pb';
	import { useConnectionContext } from '$lib/stores/connection-status.svelte';

	let clients = $state<Client[]>([]);
	let pairingRequests = $state<PairingRequest[]>([]);
	let pendingPairing = $state<Record<string, boolean>>({});
	let pendingClientAction = $state<Record<string, boolean>>({});

	const connStatus = useConnectionContext();

	let clientsConnected = false;
	let clientsBackoff = 0;
	let pairingConnected = false;
	let pairingBackoff = 0;

	const updateConnStatus = () => {
		connStatus.set(
			clientsConnected && pairingConnected,
			(!clientsConnected && clientsBackoff > 0) || (!pairingConnected && pairingBackoff > 0)
		);
	};

	onDestroy(
		ClientsStore.subscribe((update) => {
			clients = update.clients;
			clientsConnected = update.clientsConnected;
			clientsBackoff = update.clientsBackoff;
			updateConnStatus();
		})
	);
	onDestroy(
		PairingRequestsStore.subscribe((update) => {
			pairingRequests = update.requests;
			pairingConnected = update.connected;
			pairingBackoff = update.backoff;
			updateConnStatus();
		})
	);
	onDestroy(() => connStatus.reset());

	const secondsToDate = (seconds: bigint) => new Date(Number(seconds) * 1000);

	const setPending = (requestId: string, pending: boolean) => {
		pendingPairing = { ...pendingPairing, [requestId]: pending };
	};

	const setClientPending = (clientId: string, pending: boolean) => {
		pendingClientAction = { ...pendingClientAction, [clientId]: pending };
	};

	// Group the 8-digit SAS as "1234 5678" for easier comparison by eye.
	const formatSas = (sas: string) => (sas.length === 8 ? `${sas.slice(0, 4)} ${sas.slice(4)}` : sas);

	const handleApprove = async (req: PairingRequest) => {
		if (!req.sas) {
			return;
		}
		const label = req.name || req.clientId;
		// Force a deliberate comparison rather than a bare one-click confirm: the
		// admin must actively acknowledge the codes match before we release the
		// credentials.
		const matches = confirm(
			`Pair "${label}"?\n\n` +
				`Confirm this code is EXACTLY the same as the one shown on the device:\n\n` +
				`        ${formatSas(req.sas)}\n\n` +
				`Only approve if they match. If they differ, cancel and deny the request.`
		);
		if (!matches) {
			return;
		}

		setPending(req.requestId, true);
		try {
			await ApprovePairing(req.clientId, req.requestId);
		} finally {
			setPending(req.requestId, false);
		}
	};

	const handleDeny = async (req: PairingRequest) => {
		setPending(req.requestId, true);
		try {
			await DenyPairing(req.clientId, req.requestId);
		} finally {
			setPending(req.requestId, false);
		}
	};

	const handleUnpair = async (client: Client) => {
		setClientPending(client.id, true);
		try {
			await UnpairClient(client.id);
		} finally {
			setClientPending(client.id, false);
		}
	};

	const handleForget = async (client: Client) => {
		const label = client.name || client.id;
		if (!confirm(`Forget ${label}? This will not remove associated devices.`)) {
			return;
		}
		setClientPending(client.id, true);
		try {
			await ForgetClient(client.id);
		} finally {
			setClientPending(client.id, false);
		}
	};
</script>

<main class="grid gap-6">
	<section class="grid gap-4">
		<div class="flex items-center justify-between">
			<h2 class="text-lg font-semibold">Pairing Requests</h2>
			<span class="text-sm text-muted-foreground">{pairingRequests.length} pending</span>
		</div>

		{#if pairingRequests.length === 0}
			<p class="text-sm text-muted-foreground">No pairing requests.</p>
		{:else}
			<div class="grid gap-3">
				{#each pairingRequests as req (req.requestId)}
					<div class="rounded-xl border bg-card/50 p-4 shadow-sm grid gap-3">
						<div class="grid gap-1">
							<div class="flex items-center justify-between">
								<div class="font-semibold">{req.name || req.clientId}</div>
								<div class="text-xs text-muted-foreground font-mono">{req.clientId}</div>
							</div>
							{#if req.description}
								<p class="text-sm text-muted-foreground">{req.description}</p>
							{/if}
							<div class="flex flex-wrap gap-3 text-xs text-muted-foreground">
								<div class="flex items-center gap-1">
									Requested:
									<TimeSince past={secondsToDate(req.requestedAt)} />
								</div>
							</div>
						</div>

						<div class="rounded-lg border bg-muted/40 p-3 text-center">
							{#if req.sas}
								<div class="text-xs text-muted-foreground">Verify this code matches the device</div>
								<div class="mt-1 font-mono text-2xl font-semibold tracking-[0.3em]">
									{formatSas(req.sas)}
								</div>
							{:else}
								<div class="text-sm text-muted-foreground">Waiting for the device to present a code…</div>
							{/if}
						</div>

						<div class="flex gap-2">
							<Button
								class="cursor-pointer"
								disabled={!req.sas || pendingPairing[req.requestId]}
								onclick={() => handleApprove(req)}
							>
								Codes match — Confirm
							</Button>
							<Button
								class="cursor-pointer"
								variant="outline"
								disabled={pendingPairing[req.requestId]}
								onclick={() => handleDeny(req)}
							>
								Deny
							</Button>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</section>

	<section class="grid gap-4">
		<div class="flex items-center justify-between">
			<h2 class="text-lg font-semibold">Clients</h2>
			<span class="text-sm text-muted-foreground">{clients.length} total</span>
		</div>

		{#if clients.length === 0}
			<p class="text-sm text-muted-foreground">No clients registered yet.</p>
		{:else}
			<div class="grid gap-3">
				{#each clients as client (client.id)}
					<div class="rounded-xl border bg-card/50 p-4 shadow-sm grid gap-2">
						<div class="flex items-center justify-between">
							<div>
								<div class="font-semibold">{client.name || client.id}</div>
								<div class="text-xs text-muted-foreground font-mono">{client.id}</div>
							</div>
							<div class="flex items-center gap-2 text-xs">
								<span class={client.online ? 'text-green-600' : 'text-muted-foreground'}>
									{client.online ? 'Online' : 'Offline'}
								</span>
								{#if client.paired}
									<span class="text-emerald-600">Paired</span>
								{:else}
									<span class="text-muted-foreground">Unpaired</span>
								{/if}
							</div>
						</div>

						<div class="flex gap-2">
							<Button
								class="cursor-pointer"
								variant="outline"
								disabled={!client.paired || pendingClientAction[client.id]}
								onclick={() => handleUnpair(client)}
							>
								Unpair
							</Button>
							<Button
								class="cursor-pointer"
								variant="destructive"
								disabled={pendingClientAction[client.id]}
								onclick={() => handleForget(client)}
							>
								Forget
							</Button>
						</div>

						{#if client.description}
							<p class="text-sm text-muted-foreground">{client.description}</p>
						{/if}

						<div class="flex flex-wrap gap-3 text-xs text-muted-foreground">
							{#if client.firstSeen !== 0n}
								<div class="flex items-center gap-1">
									First seen:
									<TimeSince past={secondsToDate(client.firstSeen)} warn={false} />
								</div>
							{/if}
							{#if client.lastSeen !== 0n}
								<div class="flex items-center gap-1">
									Last seen:
									<TimeSince past={secondsToDate(client.lastSeen)} />
								</div>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</section>
</main>
