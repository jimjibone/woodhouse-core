<script lang="ts">
	import { Service, Service_ServiceType, Value, BoolValue, Attribute as AttributeType, BoolAttribute, IntAttribute, FloatAttribute, ColorAttribute, DurationAttribute } from '$lib/api/v1/clients/client_service_pb';
	import { Power } from 'lucide-svelte';
	import { cn } from "$lib/utils.js";
	import chroma from "chroma-js";

	export let title: string | undefined = undefined;
	export let online: boolean;
	export let service: Service;
	export let onAction: ((serviceID: string, val: Value) => Promise<void>) | undefined

	let alias: string = (title ? title + (service.alias !== "" ? ": "+service.alias : "") : service.alias);
	let attrOn: BoolAttribute | undefined
	let attrBrightness: IntAttribute | undefined
	let attrColorTemp: IntAttribute | undefined
	let attrColor: ColorAttribute | undefined
	let attrTransition: DurationAttribute | undefined
	let attrOthers: AttributeType[]

	const foregroundLight = "hsl(0 0% 100%)";
	const foregroundDark = "hsl(240 10% 3.9%)";

	let buttonForeground: string = "hsl(var(--primary-foreground) / var(--tw-text-opacity))";
	let buttonBackground: string = "rgb(250 204 21)"; // default "bg-yellow-400"

	$:{
		attrOthers = [];
		for (const attr of service.attrs) {
			if (attr.id === "on") {
				attrOn = attr.bool;
			} else if (attr.id === "brightness") {
				attrBrightness = attr.int;
			} else if (attr.id === "color_temp") {
				attrColorTemp = attr.int;
			} else if (attr.id === "color") {
				attrColor = attr.color;
			} else if (attr.id === "transition") {
				attrTransition = attr.duration;
			} else {
				attrOthers = [...attrOthers, attr];
			}
		}

		let color: any;
		const offline = !(attrOn !== undefined && attrOn.value);
		if (offline) {
			// @ts-ignore
			color = chroma.hsv(240, 4.8/100.0, 95.9/100.0);
		} else {
			if (attrColor !== undefined && attrColor.hueSat !== undefined) {
				// @ts-ignore
				color = chroma.hsv(attrColor.hueSat.hue, attrColor.hueSat.sat/100.0, 1.0);
			} else if (attrColorTemp !== undefined) {
				const kelvin = 1.0 / Number(attrColorTemp.value) * 1000000.0;
				// @ts-ignore
				color = chroma.temperature(kelvin);
			} else {
				// @ts-ignore
				color = chroma.rgb(250, 204, 21);
			}
		}
		buttonForeground = (color.luminance() < 0.5) ? foregroundLight : foregroundDark;
		buttonBackground = color.hex();
	}

	let action = async (val: Value) => {
		if (onAction) {
			onAction(service.id, val);
		}
	}

	let actionOn = async (val: boolean) => {
		action(
			new Value({
				id: "on",
				bool: new BoolValue({
					value: val
				})
			})
		);
	}

	let actionOnToggle = async () => {
		if (attrOn !== undefined) {
			actionOn(!attrOn.value);
		}
	}
</script>

{#if service.typ === Service_ServiceType.LIGHTBULB}
<!-- <div class="grid grid-cols-2 gap-4"> -->
<div class={cn("p-2 rounded-lg border bg-card text-card-foreground shadow-sm", !online && "bg-muted")}>
	<div class="flex flex-row gap-2">
		<div class="shrink">
			<div class="h-full grid place-content-center">
				{#if attrOn?.value}
				<button class={cn("p-2 rounded-full")} style="color: {buttonForeground}; background-color: {buttonBackground};" on:click={actionOnToggle}>
					<Power/>
				</button>
				{:else}
				<button class={cn("p-2 rounded-full", "bg-secondary text-secondary-foreground")} on:click={actionOnToggle}>
					<Power/>
				</button>
				{/if}
			</div>
		</div>
		<div class="grow">
			<div class="h-full flex flex-col gap-0 justify-center">
				{#if alias !== ""}
				<div class="p-0 rounded-lg">
					<p class="font-semibold">{alias}</p>
				</div>
				{/if}
				<div class="p-0 rounded-lg flex flex-row gap-2">
					{#if attrOn !== undefined}
					<p>{attrOn.value ? "On" : "Off"}</p>
					{/if}
					{#if attrBrightness !== undefined}
					<p class="text-muted-foreground">{(attrBrightness.value).toLocaleString(undefined, { maximumFractionDigits: 0 })}%</p>
					{/if}
					{#if attrColorTemp !== undefined}
					<!-- <p class="text-muted-foreground">{(1 / Number(attrColorTemp.value) * 1000000.0).toLocaleString(undefined, { maximumFractionDigits: 0 })}°K</p> -->
					<p class="text-muted-foreground">{(1 / Number(attrColorTemp.value) * 1000000.0).toFixed(0)}°K</p>
					{/if}
					{#if attrColor !== undefined}
						{#if attrColor.hueSat !== undefined}
						<p class="text-muted-foreground">Hue {(attrColor.hueSat.hue).toFixed(0)}°</p>
						<p class="text-muted-foreground">Sat {(attrColor.hueSat.sat).toFixed(0)}%</p>
						{/if}
						<!-- {#if attrColor.xy !== undefined}
						<p class="text-muted-foreground">{(attrColor.xy.x).toFixed(2)}</p>
						<p class="text-muted-foreground">{(attrColor.xy.y).toFixed(2)}</p>
						{/if} -->
					{/if}
				</div>
			</div>
		</div>
	</div>
</div>
<!-- <div class="p-4 rounded-lg shadow-lg bg-fuchsia-500">02</div>
<div class="p-4 rounded-lg shadow-lg bg-fuchsia-500">03</div>
</div> -->
{:else}
<p>ERROR Service Type {Service_ServiceType[service.typ]} is not LIGHTBULB</p>
{/if}
