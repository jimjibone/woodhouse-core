<script lang="ts">
	import { Slider } from '$lib/components/ui/slider';
	import { cn } from '$lib/utils';

	// A slider for an int attribute with a min and a max.
	let {
		name,
		value,
		min,
		max,
		step = 1n,
		onaction,
		transform = (value) => Number(value),
		units,
		invert,
		class: className = ''
	}: {
		name: string;
		value: bigint | undefined;
		min: bigint;
		max: bigint;
		step?: bigint;
		onaction: (value: bigint) => void;
		transform?: (value: bigint) => number;
		units: string;
		invert?: boolean;
		class?: string;
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

	const current = $derived(value ?? min);
	const stepNum = $derived(Number(step) || 1);
</script>

<div class={className}>{name}</div>
{#if invert}
	<Slider
		class={cn('shrink', className)}
		type="single"
		step={stepNum}
		min={-Number(max)}
		max={-Number(min)}
		value={-Number(current)}
		onValueChange={(val) => startAction(-val)}
		onValueCommit={(val) => sendAction(-val)}
	/>
{:else}
	<Slider
		class={cn('shrink', className)}
		type="single"
		step={stepNum}
		min={Number(min)}
		max={Number(max)}
		value={Number(current)}
		onValueChange={startAction}
		onValueCommit={sendAction}
	/>
{/if}
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
