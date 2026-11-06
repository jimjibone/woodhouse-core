<script lang="ts">
	import { ThermometerIcon } from '@lucide/svelte';
	import { cn } from '$lib/utils';
	import chroma from 'chroma-js';

	// The color_temp attribute is in kelvin. The pill shows a gradient from
	// warm amber at the bottom (lowest kelvin) to cool blue-white at the top
	// (highest kelvin), so dragging UP gives a cooler (higher kelvin) result.

	let {
		value,
		min,
		max,
		step = 1n,
		resetOnClose = false,
		onaction,
		class: className = ''
	}: {
		value: bigint | undefined;
		min: bigint;
		max: bigint;
		step?: bigint;
		resetOnClose?: boolean;
		onaction: (value: bigint) => void;
		class?: string;
	} = $props();

	let changing: number | null = $state(null);
	let isDragging = $state(false);

	$effect(() => {
		if (resetOnClose) {
			changing = null;
			isDragging = false;
		}
	});

	// Kelvin bounds straight from the attribute.
	const kMin = $derived(Number(min)); // warmest
	const kMax = $derived(Number(max)); // coolest
	const kStep = $derived(Number(step) || 1);
	// Keyboard moves in coarser steps than the attribute allows, as 1 K is
	// not perceptible.
	const keyStep = $derived(Math.max(kStep, 100));

	// Current kelvin value (live while dragging).
	const kelvinCurrent = $derived(changing !== null ? changing : Number(value ?? min));

	// Fill percentage from the bottom: 0 % = warmest (kMin), 100 % = coolest (kMax).
	const fillPct = $derived(
		kMax > kMin ? Math.max(0, Math.min(100, ((kelvinCurrent - kMin) / (kMax - kMin)) * 100)) : 0
	);

	const labelColor: string = $derived(
		chroma.temperature(kelvinCurrent).luminance() < 0.45 ? 'hsl(0 0% 100%)' : 'hsl(240 10% 3.9%)'
	);

	// Colour stops for the gradient background: warm amber at bottom, cool white-blue at top.
	const gradientColors = $derived(() => {
		const steps = [0, 0.1, 0.2, 0.35, 0.5, 0.65, 0.8, 0.9, 1.0];
		return steps.map((pct) => {
			// pct=0 -> bottom of pill -> warmest (kMin), pct=1 -> top -> coolest (kMax)
			const kelvin = Math.round(kMin + pct * (kMax - kMin));
			return { pct: pct * 100, color: chroma.temperature(kelvin).hex() };
		});
	});

	const gradientStyle = $derived(
		'background: linear-gradient(to top, ' +
			gradientColors()
				.map((s) => `${s.color} ${s.pct}%`)
				.join(', ') +
			');'
	);

	// Pointer -> kelvin value. The top of the pill is the coolest (kMax).
	const getValueFromPointer = (event: PointerEvent): number => {
		const el = event.currentTarget as HTMLElement;
		const rect = el.getBoundingClientRect();
		const relY = event.clientY - rect.top;
		const pct = 1 - Math.max(0, Math.min(1, relY / rect.height));
		const rawVal = pct * (kMax - kMin) + kMin;
		return Math.max(kMin, Math.min(kMax, Math.round(rawVal / kStep) * kStep));
	};

	const onpointerdown = (event: PointerEvent) => {
		event.stopPropagation();
		const el = event.currentTarget as HTMLElement;
		el.setPointerCapture(event.pointerId);
		isDragging = true;
		changing = getValueFromPointer(event);
	};

	const onpointermove = (event: PointerEvent) => {
		if (!isDragging) return;
		changing = getValueFromPointer(event);
	};

	const onpointerup = (event: PointerEvent) => {
		if (!isDragging) return;
		isDragging = false;
		if (changing !== null) {
			onaction(BigInt(changing));
		}
		changing = null;
	};

	const onkeydown = (event: KeyboardEvent) => {
		const current = Number(value ?? min);
		let newVal = current;
		// Arrow UP = cooler = higher kelvin; Arrow DOWN = warmer = lower kelvin.
		if (event.key === 'ArrowUp' || event.key === 'ArrowRight') {
			newVal = Math.min(kMax, current + keyStep);
		} else if (event.key === 'ArrowDown' || event.key === 'ArrowLeft') {
			newVal = Math.max(kMin, current - keyStep);
		} else if (event.key === 'Home') {
			newVal = kMin;
		} else if (event.key === 'End') {
			newVal = kMax;
		} else {
			return;
		}
		event.preventDefault();
		onaction(BigInt(newVal));
	};

	const formatKelvin = (k: number) => k.toLocaleString(undefined, { maximumFractionDigits: 0 }) + ' K';
</script>

<div class={cn('flex flex-col items-center gap-2', className)}>
	<!-- svelte-ignore a11y_interactive_supports_focus -->
	<div class="relative">
		<!-- External label to the left, visible only while dragging -->
		<div
			class="absolute pointer-events-none"
			style="right: calc(100% + 0.75rem); bottom: clamp(0.5rem, calc({fillPct}% - 0.75rem), calc(100% - 2rem));"
		>
			<span
				class={cn(
					'text-xl font-bold tabular-nums whitespace-nowrap transition-opacity duration-150',
					isDragging ? 'opacity-100' : 'opacity-0'
				)}
				style="color: {isDragging ? 'inherit' : 'transparent'};">{formatKelvin(kelvinCurrent)}</span
			>
		</div>

		<!-- Pill slider -->
		<div
			class="relative w-26 h-60 rounded-3xl border overflow-hidden cursor-pointer touch-none select-none focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
			role="slider"
			tabindex="0"
			aria-valuemin={kMin}
			aria-valuemax={kMax}
			aria-valuenow={kelvinCurrent}
			aria-label="Colour Temperature"
			{onpointerdown}
			{onpointermove}
			{onpointerup}
			onpointercancel={onpointerup}
			{onkeydown}
		>
			<!-- Full-height colour temperature gradient (always visible) -->
			<div class="absolute inset-0" style={gradientStyle}></div>

			<!-- Dark overlay from the top down to indicate the unselected cool region -->
			<div
				class={cn(
					'absolute top-0 left-0 right-0 bg-black/40',
					!isDragging && 'transition-[height] duration-150 ease-out'
				)}
				style="height: {100 - fillPct}%;"
			></div>

			<!-- Thermometer icon near top -->
			<div class="absolute inset-x-0 top-4 flex justify-center pointer-events-none">
				<ThermometerIcon
					class={cn('size-5 transition-opacity duration-150', fillPct > 85 ? 'text-dark' : 'text-light')}
				/>
			</div>

			<!-- Grab bar centred on the overlay/gradient boundary -->
			<div
				class={cn(
					'absolute inset-x-0 flex justify-center pointer-events-none',
					!isDragging && 'transition-[bottom] duration-150 ease-out'
				)}
				style="bottom: calc({fillPct}% - 0.8rem);"
			>
				<div class="w-10 h-1.5 rounded-full opacity-50" style="background-color: {labelColor};"></div>
			</div>

			<!-- In-pill kelvin value floating just above the grab bar, hidden while dragging -->
			<div
				class={cn(
					'absolute inset-x-0 flex justify-center pointer-events-none transition-[bottom,opacity] duration-150 ease-out',
					isDragging ? 'opacity-0' : 'opacity-100'
				)}
				style="bottom: clamp(0.75rem, calc({fillPct}% - 2.00rem), calc(100% - 3.50rem));"
			>
				<span class="text-xs font-semibold tabular-nums text-light" style="color: {labelColor};">
					{formatKelvin(kelvinCurrent)}
				</span>
			</div>
		</div>
	</div>
	<span class="text-xs text-muted-foreground">Color Temp</span>
</div>
