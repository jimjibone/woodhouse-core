<script lang="ts">
	import { onDestroy } from 'svelte';
	import Button from '$lib/components/ui/button/button.svelte';
	import TimeSince from '$lib/components/wh/ui/time-since.svelte';
	import { ClientsStore } from '$lib/stores/clients-stream';
	import { PairingRequestsStore } from '$lib/stores/pairing-requests-stream';
	import { ApprovePairing, DenyPairing, UnpairClient, BlockClient, UnblockClient } from '$lib/stores/requests';
	import type { Client, PairingRequest } from '$lib/api/v1/clients/client_pb';

	let clients = $state<Client[]>([]);
	let pairingRequests = $state<PairingRequest[]>([]);
	let pendingPairing = $state<Record<string, boolean>>({});
	let pendingClientAction = $state<Record<string, boolean>>({});

	onDestroy(ClientsStore.subscribe((update) => (clients = update.clients)));
	onDestroy(PairingRequestsStore.subscribe((update) => (pairingRequests = update.requests)));

	const secondsToDate = (seconds: bigint) => new Date(Number(seconds) * 1000);

	const setPending = (clientId: string, pending: boolean) => {
		pendingPairing = { ...pendingPairing, [clientId]: pending };
	};

	const setClientPending = (clientId: string, pending: boolean) => {
		pendingClientAction = { ...pendingClientAction, [clientId]: pending };
	};

	const handleApprove = async (req: PairingRequest) => {
		setPending(req.clientId, true);
		try {
			await ApprovePairing(req.clientId);
		} finally {
			setPending(req.clientId, false);
		}
	};

	const handleDeny = async (req: PairingRequest) => {
		setPending(req.clientId, true);
		try {
			await DenyPairing(req.clientId);
		} finally {
			setPending(req.clientId, false);
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

	const handleBlock = async (client: Client) => {
		const label = client.name || client.id;
		if (!confirm(`Block ${label}? This will invalidate tokens and block access.`)) {
			return;
		}
		setClientPending(client.id, true);
		try {
			await BlockClient(client.id);
		} finally {
			setClientPending(client.id, false);
		}
	};

	const handleUnblock = async (client: Client) => {
		const label = client.name || client.id;
		if (!confirm(`Unblock ${label}? This will restore access.`)) {
			return;
		}
		setClientPending(client.id, true);
		try {
			await UnblockClient(client.id);
		} finally {
			setClientPending(client.id, false);
		}
	};
</script>

<main class="grid gap-6">
	<section class="grid gap-4">
		<div class="flex items-center justify-between">
			<h2 class="text-lg font-semibold">Pending Pairing Requests</h2>
			<span class="text-sm text-muted-foreground">{pairingRequests.length} pending</span>
		</div>

		{#if pairingRequests.length === 0}
			<p class="text-sm text-muted-foreground">No pending pairing requests.</p>
		{:else}
			<div class="grid gap-3">
				{#each pairingRequests as req (req.clientId)}
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

						<div class="flex gap-2">
							<Button class="cursor-pointer" disabled={pendingPairing[req.clientId]} onclick={() => handleApprove(req)}>
								Approve
							</Button>
							<Button
								class="cursor-pointer"
								variant="outline"
								disabled={pendingPairing[req.clientId]}
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
								{/if}
								{#if client.blocked}
									<span class="text-red-600">Blocked</span>
								{/if}
							</div>
						</div>

						<div class="flex gap-2">
							<Button
								class="cursor-pointer"
								variant="outline"
								disabled={!client.paired || client.blocked || pendingClientAction[client.id]}
								onclick={() => handleUnpair(client)}
							>
								Unpair
							</Button>
							{#if client.blocked}
								<Button
									class="cursor-pointer"
									variant="outline"
									disabled={pendingClientAction[client.id]}
									onclick={() => handleUnblock(client)}
								>
									Unblock
								</Button>
							{:else}
								<Button
									class="cursor-pointer"
									variant="destructive"
									disabled={pendingClientAction[client.id]}
									onclick={() => handleBlock(client)}
								>
									Block
								</Button>
							{/if}
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
