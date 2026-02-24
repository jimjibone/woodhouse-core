<script lang="ts">
	import {
		BoolValueSchema,
		IntValueSchema,
		ValueSchema,
		ColorValueSchema,
		DurationValueSchema
	} from '$lib/api/v1/clients/client_service_pb';
	import type {
		Attribute,
		BoolAttribute,
		ColorAttribute,
		ColorHueSat,
		DurationAttribute,
		IntAttribute,
		Service,
		Value
	} from '$lib/api/v1/clients/client_service_pb';
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { LightbulbIcon, LightbulbOffIcon, SunIcon } from '@lucide/svelte';
	import chroma from 'chroma-js';
	import { mode } from 'mode-watcher';
	import { create } from '@bufbuild/protobuf';
	import { BoolContent, DurationContent, IntContent, FloatContent, OthersContent } from '$lib/components/wh/attributes';
	import { cn } from '$lib/utils';

	let { deviceID, online, service, ...rest }: StandardProps = $props();

	let attrOn: BoolAttribute | undefined = $state(undefined);
	let attrBrightness: IntAttribute | undefined = $state(undefined);
	let attrColorTemp: IntAttribute | undefined = $state(undefined);
	let attrColor: ColorAttribute | undefined = $state(undefined);
	let attrTransition: DurationAttribute | undefined = $state(undefined);
	let attrOthers: Attribute[] = $state([]);

	const foregroundLight = 'hsl(0 0% 100%)';
	const foregroundDark = 'hsl(240 10% 3.9%)';

	let buttonOn: boolean = $state(false);
	let buttonForeground: string = $state('hsl(var(--primary-foreground) / var(--tw-text-opacity))');
	let buttonBackground: string = $state('rgb(250 204 21)'); // default "bg-yellow-400"
	let sliderBackground: string = $state('rgb(250 204 21)'); // default "bg-yellow-400"

	$effect(() => {
		let others: Attribute[] = [];
		let onValue = false;
		let brightnessValue: BigInt | undefined = undefined;
		let colorValue: ColorHueSat | undefined = undefined;
		let colorTempValue: IntAttribute | undefined = undefined;
		for (const attr of service.attrs) {
			if (attr.id === 'on') {
				attrOn = attr.bool;
				onValue = attr.bool?.value !== undefined ? attr.bool?.value : false;
			} else if (attr.id === 'brightness') {
				attrBrightness = attr.int;
				brightnessValue = attr.int?.value !== undefined ? attr.int?.value : undefined;
			} else if (attr.id === 'color_temp') {
				attrColorTemp = attr.int;
				colorTempValue = attr.int;
			} else if (attr.id === 'color') {
				attrColor = attr.color;
				colorValue = attr.color?.hueSat !== undefined ? attr.color?.hueSat : undefined;
			} else if (attr.id === 'transition') {
				attrTransition = attr.duration;
			} else {
				others = [...others, attr];
			}
		}
		attrOthers = others;

		let color: any;
		if (!online || !onValue) {
			// Show color as if off offline.
			buttonOn = false;
			if (mode.current == 'dark') {
				color = chroma.hsl(240.06, 4.0 / 100.0, 16.0 / 100.0); // dark-muted
			} else {
				color = chroma.hsl(240, 4.8 / 100.0, 95.9 / 100.0); // light-muted
			}
		} else {
			buttonOn = true;
			if (colorValue !== undefined && colorValue !== undefined) {
				color = chroma.hsv(colorValue.hue, colorValue.sat / 100.0, Number(brightnessValue) / 100.0);
			} else if (colorTempValue !== undefined) {
				// TODO: add brightness adjustment for color temp.
				const kelvin = (1.0 / Number(colorTempValue.value)) * 1000000.0;
				color = chroma.temperature(kelvin);
			} else {
				color = chroma.rgb(250, 204, 21); // yellow
			}
		}
		buttonForeground = color.luminance() < 0.5 ? foregroundLight : foregroundDark;
		buttonBackground = color.hex();
		sliderBackground = buttonOn ? color.hex() : chroma.rgb(0, 0, 0);
	});

	let serviceAction = new ServiceAction(deviceID, service.id);
	let transition: bigint | undefined = $state(undefined);
	let drawerOpen = $state(false);

	$effect(() => {
		if (drawerOpen === false) {
			transition = undefined;
		}
	});

	const sendActionWithTransition = async (val: Value) => {
		let vals = [val];
		if (transition !== undefined) {
			vals.push(
				create(ValueSchema, {
					id: 'transition',
					duration: create(DurationValueSchema, {
						value: transition
					})
				})
			);
		}
		serviceAction.send(vals);
	};

	const sendActionOn = async (val: boolean) => {
		sendActionWithTransition(
			create(ValueSchema, {
				id: 'on',
				bool: create(BoolValueSchema, {
					value: val
				})
			})
		);
	};

	const sendActionBrightness = async (val: bigint) => {
		sendActionWithTransition(
			create(ValueSchema, {
				id: 'brightness',
				int: create(IntValueSchema, {
					value: val
				})
			})
		);
	};

	const sendActionColorTemp = async (val: bigint) => {
		sendActionWithTransition(
			create(ValueSchema, {
				id: 'color_temp',
				int: create(IntValueSchema, {
					value: val
				})
			})
		);
	};

	const sendActionColor = async (hue: number, sat: number) => {
		sendActionWithTransition(
			create(ValueSchema, {
				id: 'color',
				color: create(ColorValueSchema, {
					hueSat: {
						hue: hue,
						sat: sat
					}
				})
			})
		);
	};

	const sendActionTransition = async (val: bigint) => {
		// We don't request transitions on their own, they must be sent with a
		// light state change (e.g. change color).
		transition = val;
	};

	const oniconclick = async () => {
		if (attrOn !== undefined) {
			sendActionOn(!attrOn.value);
		}
	};

	// Vertical brightness slider state & handlers
	let brightnessChanging: number | null = $state(null);
	let brightnessIsDragging = $state(false);

	$effect(() => {
		if (drawerOpen === false) {
			brightnessChanging = null;
			brightnessIsDragging = false;
		}
	});

	const getBrightnessFromPointer = (event: PointerEvent): number => {
		if (!attrBrightness) return 0;
		const el = event.currentTarget as HTMLElement;
		const rect = el.getBoundingClientRect();
		const min = Number(attrBrightness.min);
		const max = Number(attrBrightness.max);
		const step = Number(attrBrightness.step) || 1;
		const relY = event.clientY - rect.top;
		const pct = 1 - Math.max(0, Math.min(1, relY / rect.height));
		const rawVal = pct * (max - min) + min;
		return Math.max(min, Math.min(max, Math.round(rawVal / step) * step));
	};

	const onBrightnessPointerDown = (event: PointerEvent) => {
		if (!attrBrightness) return;
		// Stop the event bubbling to the vaul Drawer, which would otherwise
		// record a pointerStart and interpret the subsequent downward drag as a
		// swipe-to-close gesture.
		event.stopPropagation();
		const el = event.currentTarget as HTMLElement;
		el.setPointerCapture(event.pointerId);
		brightnessIsDragging = true;
		brightnessChanging = getBrightnessFromPointer(event);
	};

	const onBrightnessPointerMove = (event: PointerEvent) => {
		if (!brightnessIsDragging || !attrBrightness) return;
		brightnessChanging = getBrightnessFromPointer(event);
	};

	const onBrightnessPointerUp = (event: PointerEvent) => {
		if (!brightnessIsDragging || !attrBrightness) return;
		brightnessIsDragging = false;
		if (brightnessChanging !== null) {
			sendActionBrightness(BigInt(brightnessChanging));
		}
		brightnessChanging = null;
	};

	const onBrightnessKeyDown = (event: KeyboardEvent) => {
		if (!attrBrightness) return;
		const min = Number(attrBrightness.min);
		const max = Number(attrBrightness.max);
		const step = Number(attrBrightness.step) || 1;
		const current = Number(attrBrightness.value);
		let newVal = current;
		if (event.key === 'ArrowUp' || event.key === 'ArrowRight') {
			newVal = Math.min(max, current + step);
		} else if (event.key === 'ArrowDown' || event.key === 'ArrowLeft') {
			newVal = Math.max(min, current - step);
		} else if (event.key === 'Home') {
			newVal = min;
		} else if (event.key === 'End') {
			newVal = max;
		} else {
			return;
		}
		event.preventDefault();
		sendActionBrightness(BigInt(newVal));
	};
</script>

{#snippet icon()}
	{#if buttonOn}
		<LightbulbIcon />
	{:else}
		<LightbulbOffIcon />
	{/if}
{/snippet}

{#snippet details()}
	{#if attrOn !== undefined}
		<p>{attrOn.value ? 'On' : 'Off'}</p>
	{/if}
	{#if attrBrightness !== undefined}
		<p class="text-muted-foreground">
			{attrBrightness.value.toLocaleString(undefined, { maximumFractionDigits: 0 })}%
		</p>
	{/if}
	{#if attrColorTemp !== undefined}
		<p class="text-muted-foreground">
			{(1000000.0 / Number(attrColorTemp.value)).toLocaleString(undefined, { maximumFractionDigits: 0 })}°K
		</p>
	{/if}
	{#if attrColor !== undefined && attrColor.hueSat !== undefined}
		<p class="text-muted-foreground">Hue {attrColor.hueSat.hue.toFixed(0)}°</p>
		<p class="text-muted-foreground">Sat {attrColor.hueSat.sat.toFixed(0)}%</p>
	{/if}
{/snippet}

<ServiceRoot
	{deviceID}
	{online}
	{...rest}
	{service}
	actionPending={serviceAction.pending}
	errorSignal={serviceAction.error}
	{icon}
	iconstyle="color: {buttonForeground}; background-color: {buttonBackground};"
	{oniconclick}
	{details}
	bind:drawerOpen
>
	{#if attrBrightness !== undefined}
		{@const bMin = Number(attrBrightness.min)}
		{@const bMax = Number(attrBrightness.max)}
		{@const bCurrent = brightnessChanging !== null ? brightnessChanging : Number(attrBrightness.value)}
		{@const bFillPct = Math.max(0, Math.min(100, ((bCurrent - bMin) / (bMax - bMin)) * 100))}
		<div class="flex flex-col items-center gap-2 mb-4">
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
							brightnessIsDragging ? 'opacity-100' : 'opacity-0'
						)}
						style="color: {brightnessIsDragging ? 'inherit' : 'transparent'};">{bCurrent}%</span
					>
				</div>
				<div
					class="relative w-26 h-56 rounded-3xl bg-muted border overflow-hidden cursor-pointer touch-none select-none focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
					role="slider"
					tabindex="0"
					aria-valuemin={bMin}
					aria-valuemax={bMax}
					aria-valuenow={bCurrent}
					aria-label="Brightness"
					onpointerdown={onBrightnessPointerDown}
					onpointermove={onBrightnessPointerMove}
					onpointerup={onBrightnessPointerUp}
					onpointercancel={onBrightnessPointerUp}
					onkeydown={onBrightnessKeyDown}
				>
					<!-- Fill from bottom -->
					<div
						class={cn(
							'absolute bottom-0 left-0 right-0 rounded-md',
							!brightnessIsDragging && 'transition-[height] duration-150 ease-out'
						)}
						style="height: {bFillPct}%; background-color: {sliderBackground};"
					></div>
					<!-- Sun icon near top -->
					<div class="absolute inset-x-0 top-4 flex justify-center pointer-events-none">
						<SunIcon
							class={cn('size-5 transition-colors duration-150', bFillPct > 85 ? 'opacity-90' : 'opacity-40')}
							style={bFillPct > 85 ? `color: ${buttonForeground}` : ''}
						/>
					</div>
					<!-- Grab bar centred on the fill's top edge  0.1875rem -->
					<div
						class={cn(
							'absolute inset-x-0 flex justify-center pointer-events-none',
							!brightnessIsDragging && 'transition-[bottom] duration-150 ease-out'
						)}
						style="bottom: calc({bFillPct}% - 0.8rem);"
					>
						<div class="w-10 h-1.5 rounded-full opacity-50" style="background-color: {buttonForeground}"></div>
					</div>
					<!-- Brightness value floating just above the fill top edge (hidden while dragging) -->
					<div
						class={cn(
							'absolute inset-x-0 flex justify-center pointer-events-none transition-[bottom,opacity] duration-150 ease-out',
							brightnessIsDragging ? 'opacity-0' : 'opacity-100'
						)}
						style="bottom: clamp(0.75rem, calc({bFillPct}% + 0.25rem), calc(100% - 1.75rem));"
					>
						<span
							class="text-xs font-semibold tabular-nums"
							style="color: {bFillPct > 15 ? buttonForeground : 'var(--color-muted-foreground)'};">{bCurrent}%</span
						>
					</div>
				</div>
			</div>
			<span class="text-xs text-muted-foreground">Brightness</span>
		</div>
	{/if}
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if attrOn !== undefined}
			<BoolContent name="On" attr={attrOn} onaction={sendActionOn} />
		{/if}
		{#if attrBrightness !== undefined}
			<IntContent name="Brightness" attr={attrBrightness} onaction={sendActionBrightness} units="%" />
		{/if}
		{#if attrColorTemp !== undefined}
			<IntContent
				name="Color Temp"
				attr={attrColorTemp}
				onaction={sendActionColorTemp}
				transform={(val) => 1000000.0 / Number(val)}
				units="°K"
				invert
			/>
		{/if}
		{#if attrColor !== undefined}
			{#if attrColor.hueSat !== undefined}
				<FloatContent
					name="Hue"
					value={attrColor.hueSat.hue}
					min={0}
					max={360}
					onaction={(val) => sendActionColor(val, attrColor!.hueSat!.sat)}
					units="°"
				/>
				<FloatContent
					name="Sat"
					value={attrColor.hueSat.sat}
					min={0}
					max={100}
					onaction={(val) => sendActionColor(attrColor!.hueSat!.hue, val)}
					units="%"
				/>
			{/if}
		{/if}
		{#if attrTransition !== undefined}
			<DurationContent name="Transition" attr={attrTransition} value={transition} onaction={sendActionTransition} />
		{/if}
	</div>
	<OthersContent others={attrOthers} {serviceAction} />
</ServiceRoot>
