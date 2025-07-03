<script lang="ts">
	import type { DurationAttribute } from '$lib/api/v1/clients/client_service_pb';
	import { Slider } from "$lib/components/ui/slider";
	import { cn } from '$lib/utils';

	let {
		name,
		attr,
		value,
		onaction,
		transform = (value) => Number(value)/1000.0,
		units = "s"
	}: {
		name: string,
		attr: DurationAttribute,
		value?: bigint,
		onaction: (value: bigint)=>void,
		transform?: (value: bigint)=>number,
		units?: string
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
<Slider
	class="shrink"
	type="single"
	step={Number(attr.step)}
	min={Number(attr.min)}
	max={Number(attr.max)}
	value={value ? Number(value) : Number(attr.value)}
	onValueChange={startAction}
	onValueCommit={sendAction}
/>
<div class={cn("inline-block", changing ? "font-semibold" : "text-muted-foreground")}>
	<span class="invisible block h-0 overflow-hidden font-semibold">
		{ghostMax+units}
	</span>
	<span>
		{#if changing}
			{formatNumber(transform(changing))+units}
		{:else}
			{formatNumber(transform(value ? value : attr.value))+units}
		{/if}
	</span>
</div>
