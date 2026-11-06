<script lang="ts">
	import { Slider } from '$lib/components/ui/slider';
	import { cn } from '$lib/utils';

	// A slider for a duration attribute (milliseconds) with a min and a max.
	let {
		name,
		value,
		min,
		max,
		step = 1n,
		onaction,
		transform = (value) => Number(value) / 1000.0,
		units = 's'
	}: {
		name: string;
		value: bigint | undefined;
		min: bigint;
		max: bigint;
		step?: bigint;
		onaction: (value: bigint) => void;
		transform?: (value: bigint) => number;
		units?: string;
	} = $props();

	let ghostMax: string = $derived.by(() => {
		const tmax = transform(max); // Get the transformed version of max.
		const digits = Math.abs(Math.trunc(tmax)).toString().length; // Count digits.
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
	step={Number(step) || 1}
	min={Number(min)}
	max={Number(max)}
	value={Number(value ?? min)}
	onValueChange={startAction}
	onValueCommit={sendAction}
/>
<div class={cn('inline-block', changing !== null ? 'font-semibold' : 'text-muted-foreground')}>
	<span class="invisible block h-0 overflow-hidden font-semibold">
		{ghostMax + units}
	</span>
	<span>
		{#if changing !== null}
			{formatNumber(transform(changing)) + units}
		{:else if value !== undefined}
			{formatNumber(transform(value)) + units}
		{:else}
			-
		{/if}
	</span>
</div>
