<script lang="ts">
	import {
		Service,
		Service_ServiceType,
		Value,
		BoolValue,
		Attribute as AttributeType,
		BoolAttribute,
		IntAttribute,
		ColorAttribute,
		DurationAttribute,
		IntValue,
		ActionResponse
	} from '$lib/api/v1/clients/client_service_pb';
	import { Loader, Lightbulb, LightbulbOff } from 'lucide-svelte';
	import { cn } from '$lib/utils.js';
	import chroma from 'chroma-js';

	export let title: string | undefined = undefined;
	export let online: boolean;
	export let service: Service;
	export let onAction: ((serviceID: string, vals: Value[], responseHandler: (response: ActionResponse) => void) => Promise<void>) | undefined;

	$: alias = title ? title + (service.alias !== '' ? ': ' + service.alias : '') : service.alias;
	let attrOn: BoolAttribute | undefined;
	let attrBrightness: IntAttribute | undefined;
	let attrColorTemp: IntAttribute | undefined;
	let attrColor: ColorAttribute | undefined;
	let attrTransition: DurationAttribute | undefined;
	let attrOthers: AttributeType[];
	let actionPending: boolean = false;

	const foregroundLight = 'hsl(0 0% 100%)';
	const foregroundDark = 'hsl(240 10% 3.9%)';

	let displayOn: boolean = false;
	let buttonForeground: string = 'hsl(var(--primary-foreground) / var(--tw-text-opacity))';
	let buttonBackground: string = 'rgb(250 204 21)'; // default "bg-yellow-400"

	$: {
		attrOthers = [];
		for (const attr of service.attrs) {
			if (attr.id === 'on') {
				attrOn = attr.bool;
			} else if (attr.id === 'brightness') {
				attrBrightness = attr.int;
			} else if (attr.id === 'color_temp') {
				attrColorTemp = attr.int;
			} else if (attr.id === 'color') {
				attrColor = attr.color;
			} else if (attr.id === 'transition') {
				attrTransition = attr.duration;
			} else {
				attrOthers = [...attrOthers, attr];
			}
		}

		let color: any;
		if (!online || !attrOn?.value) {
			// Show color as off if offline.
			displayOn = false;
			// @ts-ignore
			color = chroma.hsl(240, 4.8/100.0, 95.9/100.0); // light-muted
		} else {
			displayOn = true;
			if (attrColor !== undefined && attrColor.hueSat !== undefined) {
				// @ts-ignore
				color = chroma.hsv(attrColor.hueSat.hue, attrColor.hueSat.sat / 100.0, 1.0);
			} else if (attrColorTemp !== undefined) {
				const kelvin = (1.0 / Number(attrColorTemp.value)) * 1000000.0;
				// @ts-ignore
				color = chroma.temperature(kelvin);
			} else {
				// @ts-ignore
				color = chroma.rgb(250, 204, 21); // yellow
			}
		}
		buttonForeground = color.luminance() < 0.5 ? foregroundLight : foregroundDark;
		buttonBackground = color.hex();
	}

	let action = async (vals: Value[]) => {
		if (onAction) {
			actionPending = true;
			await onAction(service.id, vals, (response: ActionResponse) => {
				actionPending = false;
			});
		}
	};

	let actionOn = async (val: boolean) => {
		action([
			new Value({
				id: 'on',
				bool: new BoolValue({
					value: val
				})
			})
		]);
	};

	let actionOnToggle = async (ev: MouseEvent) => {
		ev.stopPropagation();
		if (attrOn !== undefined) {
			actionOn(!attrOn.value);
		}
	};

	let actionSetBrightness = async (ev: MouseEvent, adjustment: bigint) => {
		ev.stopPropagation();
		if (attrBrightness !== undefined) {
			let val = attrBrightness.value + adjustment;
			if (val < 0) val = 0n;
			if (val > 100) val = 100n;
			action([
				new Value({
					id: 'brightness',
					int: new IntValue({
						value: val
					})
				})
			]);
		}
	};

	import { mediaQuery } from "svelte-legos";
	import * as Dialog from "$lib/components/ui/dialog/index.js";
	import * as Drawer from "$lib/components/ui/drawer";
	import { Button } from "$lib/components/ui/button";
	import { Minus, Plus } from 'lucide-svelte';
	import { ActionResponse_ActionStatus } from '@/api/v1/clients/client_service_pb';

	const isDesktop = mediaQuery("(min-width: 768px)");
	let drawerOpen: boolean = false;
	let openDrawer = () => {
		drawerOpen = true;
	};
	let closeDrawer = () => {
		drawerOpen = false;
	};
</script>

{#if service.typ === Service_ServiceType.LIGHTBULB}
	<button
		class={cn(
			'rounded-lg border bg-card p-2 text-card-foreground shadow-sm text-left',
			!online && 'bg-muted'
		)}
		on:click={openDrawer}
	>
		<div class="flex flex-row gap-2">
			<div class="shrink">
				<div class="grid h-full place-content-center">
					{#if displayOn}
						<button
							class={cn('rounded-full p-2')}
							style="color: {buttonForeground}; background-color: {buttonBackground};"
							on:click={actionOnToggle}
						>
							{#if actionPending}
								<Loader />
							{:else}
								<Lightbulb />
							{/if}
						</button>
					{:else}
						<button
							class={cn('rounded-full p-2', 'bg-muted text-secondary-foreground')}
							on:click={actionOnToggle}
						>
							{#if actionPending}
								<Loader />
							{:else}
								<LightbulbOff />
							{/if}
						</button>
					{/if}
				</div>
			</div>
			<div class="grow">
				<div class="flex h-full flex-col justify-center gap-0">
					{#if alias !== ''}
						<div class="rounded-lg p-0">
							<p class="font-semibold">{alias}</p>
						</div>
					{/if}
					<div class="flex flex-row gap-2 rounded-lg p-0">
						{#if attrOn !== undefined}
							<p>{attrOn.value ? 'On' : 'Off'}</p>
						{/if}
						{#if attrBrightness !== undefined}
							<p class="text-muted-foreground">
								{attrBrightness.value.toLocaleString(undefined, { maximumFractionDigits: 0 })}%
							</p>
						{/if}
						{#if attrColorTemp !== undefined}
							<!-- <p class="text-muted-foreground">{(1 / Number(attrColorTemp.value) * 1000000.0).toLocaleString(undefined, { maximumFractionDigits: 0 })}°K</p> -->
							<p class="text-muted-foreground">
								{((1 / Number(attrColorTemp.value)) * 1000000.0).toFixed(0)}°K
							</p>
						{/if}
						{#if attrColor !== undefined}
							{#if attrColor.hueSat !== undefined}
								<p class="text-muted-foreground">Hue {attrColor.hueSat.hue.toFixed(0)}°</p>
								<p class="text-muted-foreground">Sat {attrColor.hueSat.sat.toFixed(0)}%</p>
							{/if}
							<!-- {#if attrColor.xy !== undefined}
						<p class="text-muted-foreground">{(attrColor.xy.x).toFixed(2)}</p>
						<p class="text-muted-foreground">{(attrColor.xy.y).toFixed(2)}</p>
						{/if} -->
						{/if}
						<!-- <Button variant="outline" builders={[builder]}>Open</Button> -->
						<!-- <Button variant="outline" on:click={chopen}>Chopen</Button> -->
					</div>
				</div>
			</div>
		</div>
	</button>

	{#if $isDesktop}
	<Dialog.Root bind:open={drawerOpen}>
		<Dialog.Content>
			<Dialog.Header>
				<Dialog.Title>{alias}</Dialog.Title>
				<Dialog.Description>
				This action cannot be undone.
				</Dialog.Description>
			</Dialog.Header>
			{#if attrBrightness !== undefined}
			<p>Brightness</p>
				<div class="p-4 pb-0">
					<div class="flex items-center justify-center space-x-2">
					<Button
						variant="outline"
						size="icon"
						class="size-12 shrink-0 rounded-full"
						on:click={(ev) => actionSetBrightness(ev, -10n)}
						disabled={attrBrightness.value <= 0}
					>
						<Minus class="size-5" />
						<span class="sr-only">Decrease</span>
					</Button>
					<div class="flex-1 text-center">
						<div class="flex justify-center content-start">
							<div class="text-4xl font-bold tracking-tighter">
								{attrBrightness.value}
								<span class="text-2xl uppercase text-muted-foreground">%</span>
							</div>
						</div>
					</div>
					<Button
						variant="outline"
						size="icon"
						class="size-12 shrink-0 rounded-full"
						on:click={(ev) => actionSetBrightness(ev, 10n)}
						disabled={attrBrightness.value >= 100}
					>
						<Plus class="size-5" />
						<span class="sr-only">Increase</span>
					</Button>
					</div>
				</div>
			{/if}
		</Dialog.Content>
	  </Dialog.Root>
	{:else}
	<Drawer.Root bind:open={drawerOpen}>
		<!-- <Drawer.Trigger asChild let:builder>
		</Drawer.Trigger> -->
		<Drawer.Content class="max-h-[96%]">
			<div class="w-full mx-auto flex flex-col overflow-auto p-4 rounded-t-[10px] ">
			<!-- <div class="min-w-96"> -->
			<Drawer.Header>
				<Drawer.Title>{alias}</Drawer.Title>
				<Drawer.Description>This action cannot be undone.</Drawer.Description>
			</Drawer.Header>
			{#if attrBrightness !== undefined}
			<div class="p-4 pb-0">
				<div class="flex items-center justify-center space-x-2">
				<Button
					variant="outline"
					size="icon"
					class="size-12 shrink-0 rounded-full"
					on:click={(ev) => actionSetBrightness(ev, -10n)}
					disabled={attrBrightness.value <= 0}
				>
					<Minus class="size-5" />
					<span class="sr-only">Decrease</span>
				</Button>
				<div class="flex-1 text-center">
					<div class="flex justify-center content-start">
						<div class="text-6xl font-bold tracking-tighter">
							{attrBrightness.value}
							<span class="text-2xl uppercase text-muted-foreground">%</span>
						</div>
					</div>
				</div>
				<Button
					variant="outline"
					size="icon"
					class="size-12 shrink-0 rounded-full"
					on:click={(ev) => actionSetBrightness(ev, 10n)}
					disabled={attrBrightness.value >= 100}
				>
					<Plus class="size-5" />
					<span class="sr-only">Increase</span>
				</Button>
				</div>
				<div class="mt-3 h-[120px]">
				<!-- <VisXYContainer {data} height={60}>
					<VisGroupedBar {x} {y} color="hsl(var(--primary) / 0.2)" />
				</VisXYContainer> -->
				</div>
			</div>

			<div class="p-4 pb-0">
				<div class="flex items-center justify-center space-x-2">
					<div class="flex-1 text-center">
						<div class="flex justify-center content-start">
							<div class="text-6xl font-bold tracking-tighter">
								{attrBrightness.value}
								<span class="text-2xl uppercase text-muted-foreground">%</span>
							</div>
						</div>
					</div>
				</div>
			</div>
			<div class="p-4 pb-0">
				<div class="flex items-center justify-center space-x-2">
					<div class="flex-1 text-center">
						<div class="flex justify-center content-start">
							<div class="text-6xl font-bold tracking-tighter">
								{attrBrightness.value}
								<span class="text-2xl uppercase text-muted-foreground">%</span>
							</div>
						</div>
					</div>
				</div>
			</div>
			<div class="p-4 pb-0">
				<div class="flex items-center justify-center space-x-2">
					<div class="flex-1 text-center">
						<div class="flex justify-center content-start">
							<div class="text-6xl font-bold tracking-tighter">
								{attrBrightness.value}
								<span class="text-2xl uppercase text-muted-foreground">%</span>
							</div>
						</div>
					</div>
				</div>
			</div>
			<div class="p-4 pb-0">
				<div class="flex items-center justify-center space-x-2">
					<div class="flex-1 text-center">
						<div class="flex justify-center content-start">
							<div class="text-6xl font-bold tracking-tighter">
								{attrBrightness.value}
								<span class="text-2xl uppercase text-muted-foreground">%</span>
							</div>
						</div>
					</div>
				</div>
			</div>
			<div class="p-4 pb-0">
				<div class="flex items-center justify-center space-x-2">
					<div class="flex-1 text-center">
						<div class="flex justify-center content-start">
							<div class="text-6xl font-bold tracking-tighter">
								{attrBrightness.value}
								<span class="text-2xl uppercase text-muted-foreground">%</span>
							</div>
						</div>
					</div>
				</div>
			</div>
			<div class="p-4 pb-0">
				<div class="flex items-center justify-center space-x-2">
					<div class="flex-1 text-center">
						<div class="flex justify-center content-start">
							<div class="text-6xl font-bold tracking-tighter">
								{attrBrightness.value}
								<span class="text-2xl uppercase text-muted-foreground">%</span>
							</div>
						</div>
					</div>
				</div>
			</div>
			<div class="p-4 pb-0">
				<div class="flex items-center justify-center space-x-2">
					<div class="flex-1 text-center">
						<div class="flex justify-center content-start">
							<div class="text-6xl font-bold tracking-tighter">
								{attrBrightness.value}
								<span class="text-2xl uppercase text-muted-foreground">%</span>
							</div>
						</div>
					</div>
				</div>
			</div>
			<div class="p-4 pb-0">
				<div class="flex items-center justify-center space-x-2">
					<div class="flex-1 text-center">
						<div class="flex justify-center content-start">
							<div class="text-6xl font-bold tracking-tighter">
								{attrBrightness.value}
								<span class="text-2xl uppercase text-muted-foreground">%</span>
							</div>
						</div>
					</div>
				</div>
			</div>
			<div class="p-4 pb-0">
				<div class="flex items-center justify-center space-x-2">
					<div class="flex-1 text-center">
						<div class="flex justify-center content-start">
							<div class="text-6xl font-bold tracking-tighter">
								{attrBrightness.value}
								<span class="text-2xl uppercase text-muted-foreground">%</span>
							</div>
						</div>
					</div>
				</div>
			</div>
			<div class="p-4 pb-0">
				<div class="flex items-center justify-center space-x-2">
					<div class="flex-1 text-center">
						<div class="flex justify-center content-start">
							<div class="text-6xl font-bold tracking-tighter">
								{attrBrightness.value}
								<span class="text-2xl uppercase text-muted-foreground">%</span>
							</div>
						</div>
					</div>
				</div>
			</div>
			{/if}
			<Drawer.Footer>
				<Drawer.Close>Cancel</Drawer.Close>
			</Drawer.Footer>
			<!-- </div> -->
			</div>
		</Drawer.Content>
	</Drawer.Root>
	{/if}
{:else}
	<p>ERROR Service Type {Service_ServiceType[service.typ]} is not LIGHTBULB</p>
{/if}
