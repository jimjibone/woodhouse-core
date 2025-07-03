<script lang="ts">
	import { Slider } from "$lib/components/ui/slider";
	import { cn } from '$lib/utils';

	let {
		name,
		value,
		min,
		max,
		step = 1,
		onaction,
		transform = (value) => value,
		maximumFractionDigits = 0,
		units
	}: {
		name: string,
		value: number,
		min: number,
		max: number,
		step?: number,
		onaction: (value: number)=>void,
		transform?: (value: number)=>number,
		maximumFractionDigits?: number,
		units: string
	} = $props();

	let ghostMax: string = $derived.by(() => {
		const tmax = transform(max); // Get the transformed version of max.
		const digits = Math.abs(Math.trunc(tmax)).toString().length; // Count digits.
		const ghost = formatNumber(Number('9'.repeat(digits))); // Generate number like 9,999 (if digits = 4).
		return ghost;
	});

	let formatNumber = (val: number) => {
		return val.toLocaleString(undefined, { maximumFractionDigits: maximumFractionDigits });
	};

	let changing: number | null = $state(null);

	const startAction = (val: number) => {
		changing = val;
	};

	const sendAction = async (val: number) => {
		changing = null;
		onaction(val);
	};
</script>

<div>{name}</div>
<Slider
	class="shrink"
	type="single"
	step={step}
	min={min}
	max={max}
	value={value}
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
			{formatNumber(transform(value))+units}
		{/if}
	</span>
</div>
