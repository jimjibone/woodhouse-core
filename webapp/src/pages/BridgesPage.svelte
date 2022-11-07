<script lang="ts">
	import { formatISO9075, fromUnixTime } from 'date-fns';
	import { onMount, onDestroy } from 'svelte/internal';
    import type { Unsubscriber } from 'svelte/store';
	import type { BridgeInfo } from '../api/bridge_pb';
	import { bridgeInfosStream } from '../stores/bridges';

	let bridgeInfos: BridgeInfo[] = [];
	let bridgeInfosConnected: boolean = false;
	let unsubscribeBridgeInfos: Unsubscriber = null;
	let unsubscribeBridgeInfosConnected: Unsubscriber = null;

	onMount(async () => {
		unsubscribeBridgeInfos = bridgeInfosStream.subscribeData(value => { bridgeInfos = value; });
		unsubscribeBridgeInfosConnected = bridgeInfosStream.subscribeConnected(value => { bridgeInfosConnected = value; });
	});

	onDestroy(() => {
		unsubscribeBridgeInfos();
		unsubscribeBridgeInfosConnected();
	});
</script>

<section class="hero">
	<div class="hero-body">
		<p class="title">
			Bridges
		</p>
		<p class="subtitle">
			{#if bridgeInfosConnected}
				Connected - {bridgeInfos.length} {bridgeInfos.length == 1 ? "bridge" : "bridges"}
			{:else}
				Disconnected
			{/if}
		</p>
	</div>
</section>
{#each bridgeInfos as info (info.getBridgeId())}
<div class="column is-full">
	<div class="box">
		{info.getBridgeId()} {info.getName()} {info.getDescription()} {formatISO9075(fromUnixTime(info.getBootTime().getSeconds()))}
	</div>
</div>
{:else}
<div class="column is-full">
	<div class="box">
		No bridges.
	</div>
</div>
{/each}
