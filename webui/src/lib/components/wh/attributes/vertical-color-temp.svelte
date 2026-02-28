<script lang="ts">
	import type { IntAttribute } from '$lib/api/v1/clients/client_service_pb';
	import { ThermometerIcon } from '@lucide/svelte';
	import { cn } from '$lib/utils';
	import chroma from 'chroma-js';

	// The colour_temp attribute is stored in mireds (1_000_000 / kelvin).
	// Min mireds = coolest (highest kelvin), max mireds = warmest (lowest kelvin).
	// The slider is inverted so that dragging UP gives a warmer (higher kelvin) result,
	// matching the visual gradient (warm amber at bottom → cool blue-white at top).

	let {
		attr,
		resetOnClose = false,
		onaction
	}: {
		attr: IntAttribute;
		resetOnClose?: boolean;
		onaction: (value: bigint) => void;
	} = $props();

	let changing: number | null = $state(null);
	let isDragging = $state(false);

	$effect(() => {
		if (resetOnClose) {
			changing = null;
			isDragging = false;
		}
	});

	// Mired bounds straight from the attribute.
	const miredMin = $derived(Number(attr.min)); // coolest  (e.g. 153 ≈ 6500 K)
	const miredMax = $derived(Number(attr.max)); // warmest  (e.g. 500 ≈ 2000 K)
	const miredStep = $derived(Number(attr.step) || 1);

	// Current mired value (live while dragging).
	const miredCurrent = $derived(changing !== null ? changing : Number(attr.value));

	// Kelvin for display.
	const kelvinCurrent = $derived(Math.round(1_000_000 / miredCurrent));

	// Fill percentage: 0 % = coolest (miredMin), 100 % = warmest (miredMax).
	// Dragging UP → lower fill % → cooler → lower mired value.
	// We invert so the pill fills from the bottom with warm colour.
	const fillPct = $derived(Math.max(0, Math.min(100, ((miredCurrent - miredMax) / (miredMin - miredMax)) * 100)));

	const labelColor: string = $derived(
		chroma.temperature(kelvinCurrent).luminance() < 0.45 ? 'hsl(0 0% 100%)' : 'hsl(240 10% 3.9%)'
	);

	// Colour stops for the gradient background: warm amber at bottom, cool white-blue at top.
	const gradientColors = $derived(() => {
		const steps = [0, 0.1, 0.2, 0.35, 0.5, 0.65, 0.8, 0.9, 1.0];
		return steps.map((pct) => {
			// pct=0 → bottom of pill → warmest (miredMax), pct=1 → top → coolest (miredMin)
			const mireds = miredMax - pct * (miredMax - miredMin);
			const kelvin = Math.round(1_000_000 / mireds);
			return { pct: pct * 100, color: chroma.temperature(kelvin).hex() };
		});
	});

	// Build gradient from bottom (warm) to top (cool) using chroma-computed stops.
	const gradientStyle = $derived(
		'background: linear-gradient(to top, ' +
			gradientColors()
				.map((s) => `${s.color} ${s.pct}%`)
				.join(', ') +
			');'
	);

	// The "needle" colour that sits on the gradient at the current position —
	// used for the grab bar and in-pill label to contrast with the gradient.
	const needleColor = $derived(() => {
		// Interpolate: warm end → dark text, cool end → dark text, middle → dark.
		// The gradient is always light-ish so a semi-transparent dark bar works well.
		return 'rgba(0,0,0,0.55)';
	});

	// Pointer → mired value, keeping the visual inversion.
	// Dragging to the TOP of the pill = coolest = miredMin.
	const getValueFromPointer = (event: PointerEvent): number => {
		const el = event.currentTarget as HTMLElement;
		const rect = el.getBoundingClientRect();
		const relY = event.clientY - rect.top;
		// pct = 0 at top (cool), 1 at bottom (warm) → maps to miredMin..miredMax
		const pct = Math.max(0, Math.min(1, relY / rect.height));
		const rawVal = pct * (miredMax - miredMin) + miredMin;
		return Math.max(miredMin, Math.min(miredMax, Math.round(rawVal / miredStep) * miredStep));
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
		const current = Number(attr.value);
		let newVal = current;
		// Arrow UP = warmer = higher mired; Arrow DOWN = cooler = lower mired.
		if (event.key === 'ArrowUp' || event.key === 'ArrowRight') {
			newVal = Math.min(miredMax, current + miredStep);
		} else if (event.key === 'ArrowDown' || event.key === 'ArrowLeft') {
			newVal = Math.max(miredMin, current - miredStep);
		} else if (event.key === 'Home') {
			newVal = miredMin;
		} else if (event.key === 'End') {
			newVal = miredMax;
		} else {
			return;
		}
		event.preventDefault();
		onaction(BigInt(newVal));
	};

	const formatKelvin = (k: number) => k.toLocaleString(undefined, { maximumFractionDigits: 0 }) + '°K';
</script>

<div class="flex flex-col items-center gap-2">
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
			aria-valuemin={miredMin}
			aria-valuemax={miredMax}
			aria-valuenow={miredCurrent}
			aria-label="Colour Temperature"
			{onpointerdown}
			{onpointermove}
			{onpointerup}
			onpointercancel={onpointerup}
			{onkeydown}
		>
			<!-- Full-height colour temperature gradient (always visible) -->
			<div class="absolute inset-0" style={gradientStyle}></div>

			<!-- Dark overlay from the top down to indicate the "unselected" cool region -->
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

			<!-- In-pill °K value floating just above the grab bar, hidden while dragging -->
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
