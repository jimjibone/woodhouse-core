<script lang="ts">
	import { Slider } from '$lib/components/ui/slider';
	import { cn } from '$lib/utils';

	// A slider for a float attribute with a min and a max.
	let {
		name,
		value,
		min,
		max,
		step = 1,
		onaction,
		transform = (value) => value,
		maximumFractionDigits = 0,
		units,
		class: className = ''
	}: {
		name: string;
		value: number | undefined;
		min: number;
		max: number;
		step?: number;
		onaction: (value: number) => void;
		transform?: (value: number) => number;
		maximumFractionDigits?: number;
		units: string;
		class?: string;
	} = $props();

	let ghostMax: string = $derived.by(() => {
		const tmax = transform(max); // Get the transformed version of max.
		const digits = Math.abs(Math.trunc(tmax)).toString().length; // Count digits.
		const ghost = formatNumber(Number('9'.repeat(digits))); // Generate number like 9,999 (if digits = 4).
		return ghost;
	});

	let formatNumber = (val: number) => {
		return val.toLocaleString(undefined, {
			minimumFractionDigits: maximumFractionDigits,
			maximumFractionDigits: maximumFractionDigits
		});
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

<div class={className}>{name}</div>
<Slider
	class={cn('shrink', className)}
	type="single"
	{step}
	{min}
	{max}
	value={value ?? min}
	onValueChange={startAction}
	onValueCommit={sendAction}
/>
<div class={cn('inline-block', changing !== null ? 'font-semibold' : 'text-muted-foreground', className)}>
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
