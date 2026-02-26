<script lang="ts">
	import type { IntAttribute } from '$lib/api/v1/clients/client_service_pb';
	import { SunIcon } from '@lucide/svelte';
	import { cn } from '$lib/utils';

	let {
		attr,
		fillColor,
		labelColor,
		onaction,
		resetOnClose = false
	}: {
		attr: IntAttribute;
		fillColor: string;
		labelColor: string;
		onaction: (value: bigint) => void;
		resetOnClose?: boolean;
	} = $props();

	let changing: number | null = $state(null);
	let isDragging = $state(false);

	$effect(() => {
		if (resetOnClose) {
			changing = null;
			isDragging = false;
		}
	});

	const bMin = $derived(Number(attr.min));
	const bMax = $derived(Number(attr.max));
	const bStep = $derived(Number(attr.step) || 1);
	const bCurrent = $derived(changing !== null ? changing : Number(attr.value));
	const bFillPct = $derived(Math.max(0, Math.min(100, ((bCurrent - bMin) / (bMax - bMin)) * 100)));

	const getValueFromPointer = (event: PointerEvent): number => {
		const el = event.currentTarget as HTMLElement;
		const rect = el.getBoundingClientRect();
		const relY = event.clientY - rect.top;
		const pct = 1 - Math.max(0, Math.min(1, relY / rect.height));
		const rawVal = pct * (bMax - bMin) + bMin;
		return Math.max(bMin, Math.min(bMax, Math.round(rawVal / bStep) * bStep));
	};

	const onpointerdown = (event: PointerEvent) => {
		// Stop the event bubbling to the vaul Drawer, which would otherwise
		// record a pointerStart and interpret the subsequent downward drag as a
		// swipe-to-close gesture.
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
		if (event.key === 'ArrowUp' || event.key === 'ArrowRight') {
			newVal = Math.min(bMax, current + bStep);
		} else if (event.key === 'ArrowDown' || event.key === 'ArrowLeft') {
			newVal = Math.max(bMin, current - bStep);
		} else if (event.key === 'Home') {
			newVal = bMin;
		} else if (event.key === 'End') {
			newVal = bMax;
		} else {
			return;
		}
		event.preventDefault();
		onaction(BigInt(newVal));
	};
</script>

<div class="flex flex-col items-center gap-2">
	<!-- svelte-ignore a11y_interactive_supports_focus -->
	<div class="relative">
		<!-- External label to the left, visible only while dragging -->
		<div
			class="absolute pointer-events-none"
			style="right: calc(100% + 0.75rem); bottom: clamp(0.5rem, calc({bFillPct}% - 0.75rem), calc(100% - 2rem));"
		>
			<span
				class={cn(
					'text-xl font-bold tabular-nums whitespace-nowrap transition-opacity duration-150',
					isDragging ? 'opacity-100' : 'opacity-0'
				)}
				style="color: {isDragging ? 'inherit' : 'transparent'};">{bCurrent}%</span
			>
		</div>

		<!-- Pill slider -->
		<div
			class="relative w-26 h-60 rounded-3xl bg-muted border overflow-hidden cursor-pointer touch-none select-none focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
			role="slider"
			tabindex="0"
			aria-valuemin={bMin}
			aria-valuemax={bMax}
			aria-valuenow={bCurrent}
			aria-label="Brightness"
			{onpointerdown}
			{onpointermove}
			{onpointerup}
			onpointercancel={onpointerup}
			{onkeydown}
		>
			<!-- Fill from bottom -->
			<div
				class={cn('absolute bottom-0 left-0 right-0', !isDragging && 'transition-[height] duration-150 ease-out')}
				style="height: {bFillPct}%; background-color: {fillColor};"
			></div>

			<!-- Sun icon near top -->
			<div class="absolute inset-x-0 top-4 flex justify-center pointer-events-none">
				<SunIcon
					class={cn('size-5 transition-colors duration-150', bFillPct > 85 ? 'opacity-90' : 'opacity-40')}
					style={bFillPct > 85 ? `color: ${labelColor}` : ''}
				/>
			</div>

			<!-- Grab bar centred on the fill's top edge -->
			<div
				class={cn(
					'absolute inset-x-0 flex justify-center pointer-events-none',
					!isDragging && 'transition-[bottom] duration-150 ease-out'
				)}
				style="bottom: calc({bFillPct}% - 0.8rem);"
			>
				<div class="w-10 h-1.5 rounded-full opacity-50" style="background-color: {labelColor};"></div>
			</div>

			<!-- In-pill value, floating just above fill top edge, hidden while dragging -->
			<div
				class={cn(
					'absolute inset-x-0 flex justify-center pointer-events-none transition-[bottom,opacity] duration-150 ease-out',
					isDragging ? 'opacity-0' : 'opacity-100'
				)}
				style="bottom: clamp(0.75rem, calc({bFillPct}% + 0.25rem), calc(100% - 1.75rem));"
			>
				<span
					class="text-xs font-semibold tabular-nums"
					style="color: {bFillPct > 15 ? labelColor : 'var(--color-muted-foreground)'};">{bCurrent}%</span
				>
			</div>
		</div>
	</div>
	<span class="text-xs text-muted-foreground">Brightness</span>
</div>
