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
	import { LightbulbIcon, LightbulbOffIcon } from '@lucide/svelte';
	import chroma from 'chroma-js';
	import { mode } from 'mode-watcher';
	import { create } from '@bufbuild/protobuf';
	import { BoolContent, DurationContent, IntContent, FloatContent, OthersContent } from '$lib/components/wh/attributes';
	import VerticalBrightnessContent from '$lib/components/wh/attributes/vertical-brightness.svelte';

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
			buttonOn = online && onValue;
			if (mode.current == 'dark') {
				color = chroma.hsl(240.06, 4.0 / 100.0, 16.0 / 100.0); // dark-muted
			} else {
				color = chroma.hsl(240, 4.8 / 100.0, 95.9 / 100.0); // light-muted
			}
		} else {
			buttonOn = true;
			if (brightnessValue == 0n) {
				color = chroma.rgb(0, 0, 0);
			} else {
				const bri = Number(brightnessValue) / 200.0 + 0.5;
				if (colorValue !== undefined && colorValue !== undefined) {
					color = chroma.hsv(colorValue.hue, colorValue.sat / 100.0, bri);
				} else if (colorTempValue !== undefined) {
					const kelvin = (1.0 / Number(colorTempValue.value)) * 1000000.0;
					const ct = chroma.temperature(kelvin).hsv();
					color = chroma.hsv(ct[0], ct[1], ct[2] * bri);
				} else {
					color = chroma.hsv(34, 0.75, bri); // yellow
				}
			}
		}
		buttonForeground = color.luminance() < 0.5 ? foregroundLight : foregroundDark;
		buttonBackground = color.hex();
		sliderBackground = buttonOn ? color.hex() : chroma.rgb(0, 0, 0).hex();
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
		<div class="mb-4 flex justify-center">
			<VerticalBrightnessContent
				attr={attrBrightness}
				fillColor={sliderBackground}
				labelColor={buttonForeground}
				onaction={sendActionBrightness}
				resetOnClose={!drawerOpen}
			/>
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
