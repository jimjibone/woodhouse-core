<script lang="ts">
	import type { IntAttribute } from '$lib/api/v1/clients/client_service_pb';
	import { Slider } from "$lib/components/ui/slider";
	import { cn } from '$lib/utils';

	let {
		name,
		attr,
		onaction,
		transform = (value) => Number(value),
		units,
		invert
	}: {
		name: string,
		attr: IntAttribute,
		onaction: (value: bigint)=>void,
		transform?: (value: bigint)=>number,
		units: string,
		invert?: boolean
	} = $props();

	let ghostMax: string = $derived.by(() => {
		const max = transform(attr.max); // Get the transformed version of max.
		const digits = Math.abs(Math.trunc(max)).toString().length; // Count digits.
		const ghost = formatNumber(Number('9'.repeat(digits))); // Generate number like 9,999 (if digits = 4).
		return ghost;
	});

	let formatNumber = (val: number) => {
		return val.toLocaleString(undefined, { maximumFractionDigits: 0 });
	};

	let changing: bigint | null = $state(null);

	const startAction = (val: number) => {
		changing = BigInt(val);
	};

	const sendAction = async (val: number) => {
		changing = null;
		onaction(BigInt(val));
	};
</script>

<div>{name}</div>
{#if invert}
	<Slider
		class="shrink"
		type="single"
		step={Number(attr.step)}
		min={-Number(attr.max)}
		max={-Number(attr.min)}
		value={-Number(attr.value)}
		onValueChange={(val) => startAction(-val)}
		onValueCommit={(val) => sendAction(-val)}
	/>
{:else}
	<Slider
		class="shrink"
		type="single"
		step={Number(attr.step)}
		min={Number(attr.min)}
		max={Number(attr.max)}
		value={Number(attr.value)}
		onValueChange={startAction}
		onValueCommit={sendAction}
	/>
{/if}
<div class={cn("inline-block", changing ? "font-semibold" : "text-muted-foreground")}>
	<span class="invisible block h-0 overflow-hidden font-semibold">
		{ghostMax+units}
	</span>
	<span>
		{#if changing}
			{formatNumber(transform(changing))+units}
		{:else}
			{formatNumber(transform(attr.value))+units}
		{/if}
	</span>
</div>
