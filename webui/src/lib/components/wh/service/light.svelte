<script lang="ts">
	import ServiceRoot, { type StandardProps } from './service-root.svelte';
	import ServiceAction from './service-action.svelte';
	import { LightbulbIcon, LightbulbOffIcon } from '@lucide/svelte';
	import chroma from 'chroma-js';
	import { mode } from 'mode-watcher';
	import type { Value } from '$lib/api/v2/clients/client_service_pb';
	import {
		BoolContent,
		DurationContent,
		IntContent,
		FloatContent,
		OthersContent,
		VerticalBrightnessContent,
		VerticalColorTempContent
	} from '$lib/components/wh/attributes';
	import {
		asColor,
		attrBool,
		attrEnum,
		attrInt,
		boolValue,
		colorHueSatValue,
		durationValue,
		field,
		findAttr,
		findCommand,
		intValue,
		specRange
	} from '$lib/device';
	import { Light } from '$lib/schema/zz_services';

	let { deviceID, endpointID, online, service, ...rest }: StandardProps = $props();

	const on = $derived(attrBool(service, Light.attributes.on));
	const brightness = $derived(attrInt(service, Light.attributes.brightness));
	const colorTemp = $derived(attrInt(service, Light.attributes.colorTemp));
	const color = $derived(asColor(findAttr(service, Light.attributes.color)?.value));
	// Active colour mode ('color_temp' or 'color'), undefined when the light
	// does not report one.
	const colorMode = $derived(attrEnum(service, Light.attributes.colorMode));
	const dimColorTemp = $derived(colorMode === 'color');
	const dimColor = $derived(colorMode === 'color_temp');

	const brightnessAttr = $derived(findAttr(service, Light.attributes.brightness));
	const colorTempAttr = $derived(findAttr(service, Light.attributes.colorTemp));
	const colorAttr = $derived(findAttr(service, Light.attributes.color));
	// Hue and saturation are editable with the hs feature; xy is shown read-only.
	const hasHS = $derived(service.features.includes(Light.features.hs) && colorAttr !== undefined);

	// The attribute spec carries the range, narrowed by the bridge to what the
	// light supports.
	const bounds = (spec: typeof brightnessAttr, defMin: number, defMax: number) => {
		const r = specRange(spec?.spec);
		return {
			min: BigInt(r.min ?? defMin),
			max: BigInt(r.max ?? defMax),
			step: BigInt(Math.max(1, r.step ?? 1))
		};
	};
	const brightnessBounds = $derived(bounds(brightnessAttr, 0, 100));
	const colorTempBounds = $derived(bounds(colorTempAttr, 1000, 10000));

	// A transition can only go with the set command.
	const setCommand = $derived(findCommand(service, Light.commands.set));
	const canTransition = $derived(setCommand?.params.some((p) => p.id === 'transition') ?? false);
	const transitionSpec = $derived(setCommand?.params.find((p) => p.id === 'transition')?.spec);
	const transitionBounds = $derived.by(() => {
		const r = specRange(transitionSpec);
		return {
			min: BigInt(r.min ?? 0),
			max: BigInt(r.max ?? 300_000),
			step: BigInt(Math.max(1, r.step ?? 1000))
		};
	});

	const foregroundLight = 'hsl(0 0% 100%)';
	const foregroundDark = 'hsl(240 10% 3.9%)';

	// The colours of the icon button and the brightness slider follow what the
	// light is showing.
	const looks = $derived.by(() => {
		const lit = online && on === true;
		let col: chroma.Color;
		if (!lit) {
			// Show as off when offline.
			if (mode.current === 'dark') {
				col = chroma.hsl(240.06, 4.0 / 100.0, 16.0 / 100.0); // dark-muted
			} else {
				col = chroma.hsl(240, 4.8 / 100.0, 95.9 / 100.0); // light-muted
			}
		} else if (brightness === 0n) {
			col = chroma.rgb(0, 0, 0);
		} else {
			const bri = (brightness === undefined ? 100 : Number(brightness)) / 200.0 + 0.5;
			// An explicit colour mode picks the branch; otherwise colour wins
			// over colour temp.
			const useColorTemp =
				colorTemp !== undefined &&
				(colorMode === 'color_temp' || (colorMode !== 'color' && color?.hueSat === undefined));
			if (color?.hueSat !== undefined && !useColorTemp) {
				col = chroma.hsv(color.hueSat.hue, color.hueSat.sat / 100.0, bri);
			} else if (colorTemp !== undefined) {
				const ct = chroma.temperature(Number(colorTemp)).hsv();
				col = chroma.hsv(ct[0], ct[1], ct[2] * bri);
			} else {
				col = chroma.hsv(34, 0.75, bri); // yellow
			}
		}
		return {
			lit,
			foreground: col.luminance() < 0.5 ? foregroundLight : foregroundDark,
			background: col.hex(),
			slider: lit ? col.hex() : chroma.rgb(0, 0, 0).hex()
		};
	});

	// The IDs are fixed for the life of the component.
	// svelte-ignore state_referenced_locally
	const serviceAction = new ServiceAction(deviceID, endpointID, service.id);
	let transition: bigint | undefined = $state(undefined);
	let drawerOpen = $state(false);

	$effect(() => {
		if (drawerOpen === false) {
			transition = undefined;
		}
	});

	// A change goes as a plain write, or, when a transition was chosen, as the
	// set command so the transition travels with it.
	const send = (id: string, value: Value) => {
		if (canTransition && transition !== undefined) {
			serviceAction.command(Light.commands.set, [field(id, value), field('transition', durationValue(transition))]);
		} else {
			serviceAction.writeValue(id, value);
		}
	};

	const sendOn = (val: boolean) => send(Light.attributes.on, boolValue(val));
	const sendBrightness = (val: bigint) => send(Light.attributes.brightness, intValue(val));
	const sendColorTemp = (val: bigint) => send(Light.attributes.colorTemp, intValue(val));
	const sendColor = (hue: number, sat: number) => send(Light.attributes.color, colorHueSatValue(hue, sat));

	// Transitions are not requested on their own, they go with the next
	// change (e.g. a colour change).
	const setTransition = (val: bigint) => {
		transition = val;
	};

	const oniconclick = () => {
		if (on !== undefined) {
			sendOn(!on);
		}
	};
</script>

{#snippet icon()}
	{#if looks.lit}
		<LightbulbIcon />
	{:else}
		<LightbulbOffIcon />
	{/if}
{/snippet}

{#snippet details()}
	{#if on !== undefined}
		<p>{on ? 'On' : 'Off'}</p>
	{/if}
	{#if brightness !== undefined}
		<p class="text-muted-foreground">
			{brightness.toLocaleString(undefined, { maximumFractionDigits: 0 })}%
		</p>
	{/if}
	{#if colorTemp !== undefined && colorMode !== 'color'}
		<p class="text-muted-foreground">
			{colorTemp.toLocaleString(undefined, { maximumFractionDigits: 0 })} K
		</p>
	{/if}
	{#if color?.hueSat !== undefined && colorMode !== 'color_temp'}
		<p class="text-muted-foreground">Hue {color.hueSat.hue.toFixed(0)}&deg;</p>
		<p class="text-muted-foreground">Sat {color.hueSat.sat.toFixed(0)}%</p>
	{/if}
{/snippet}

<ServiceRoot
	{deviceID}
	{endpointID}
	{online}
	{...rest}
	{service}
	actionPending={serviceAction.pending}
	errorSignal={serviceAction.error}
	{icon}
	iconstyle="color: {looks.foreground}; background-color: {looks.background};"
	{oniconclick}
	{details}
	bind:drawerOpen
>
	{#if brightnessAttr !== undefined || colorTempAttr !== undefined}
		<div class="mb-4 flex justify-center gap-4">
			{#if brightnessAttr !== undefined}
				<VerticalBrightnessContent
					value={brightness}
					min={brightnessBounds.min}
					max={brightnessBounds.max}
					step={brightnessBounds.step}
					fillColor={looks.slider}
					onaction={sendBrightness}
					resetOnClose={!drawerOpen}
				/>
			{/if}
			{#if colorTempAttr !== undefined}
				<VerticalColorTempContent
					value={colorTemp}
					min={colorTempBounds.min}
					max={colorTempBounds.max}
					step={colorTempBounds.step}
					onaction={sendColorTemp}
					resetOnClose={!drawerOpen}
					class={dimColorTemp ? 'opacity-50' : ''}
				/>
			{/if}
		</div>
	{/if}
	<div class="grid grid-cols-[auto_1fr_auto] gap-4 items-center">
		{#if on !== undefined}
			<BoolContent name="On" value={on} onaction={sendOn} />
		{/if}
		{#if brightnessAttr !== undefined}
			<IntContent
				name="Brightness"
				value={brightness}
				min={brightnessBounds.min}
				max={brightnessBounds.max}
				step={brightnessBounds.step}
				onaction={sendBrightness}
				units="%"
			/>
		{/if}
		{#if colorTempAttr !== undefined}
			<IntContent
				name="Color Temp"
				value={colorTemp}
				min={colorTempBounds.min}
				max={colorTempBounds.max}
				step={colorTempBounds.step}
				onaction={sendColorTemp}
				units=" K"
				class={dimColorTemp ? 'opacity-50' : ''}
			/>
		{/if}
		{#if hasHS}
			<FloatContent
				name="Hue"
				value={color?.hueSat?.hue}
				min={0}
				max={360}
				onaction={(val) => sendColor(val, color?.hueSat?.sat ?? 100)}
				units="&deg;"
				class={dimColor ? 'opacity-50' : ''}
			/>
			<FloatContent
				name="Sat"
				value={color?.hueSat?.sat}
				min={0}
				max={100}
				onaction={(val) => sendColor(color?.hueSat?.hue ?? 0, val)}
				units="%"
				class={dimColor ? 'opacity-50' : ''}
			/>
		{/if}
		{#if canTransition}
			<DurationContent
				name="Transition"
				value={transition}
				min={transitionBounds.min}
				max={transitionBounds.max}
				step={transitionBounds.step}
				onaction={setTransition}
			/>
		{/if}
	</div>
	<!-- xy colour is read-only, so it stays with the generic attributes. -->
	<OthersContent
		{deviceID}
		{endpointID}
		{service}
		action={serviceAction}
		exclude={[
			Light.attributes.on,
			Light.attributes.brightness,
			Light.attributes.colorTemp,
			...(hasHS ? [Light.attributes.color] : [])
		]}
		excludeCommands={[Light.commands.set]}
	/>
</ServiceRoot>
