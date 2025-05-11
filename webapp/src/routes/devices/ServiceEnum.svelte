<script lang="ts">
	import { tick } from "svelte";
	import { ServiceRoot } from '$lib/components/wh/service'
	import { Rows3, Check, ChevronsUpDown } from 'lucide-svelte';
	import { cn } from "$lib/utils.js";

	import * as Command from "$lib/components/ui/command/index.js";
	import * as Popover from "$lib/components/ui/popover/index.js";
	import { Button } from "$lib/components/ui/button/index.js";

	import {
		Service,
		Service_ServiceType,
		EnumAttribute,
		EnumValue,
		Value,
	} from '$lib/api/v1/clients/client_service_pb';

	export let title: string | undefined = undefined;
	export let online: boolean;
	export let service: Service;
	export let onAction: ((serviceID: string, vals: Value[]) => Promise<void>) | undefined;

	let attrValue: EnumAttribute | undefined;
	let showOptions: boolean = false;

	$: {
		for (const attr of service.attrs) {
			if (attr.id === 'value') {
				attrValue = attr.enum;
			}
		}
	}

	let action = async (val: string) => {
		if (onAction) {
			// actionPending = true;
			await onAction(service.id, [
				new Value({
					id: 'value',
					enum: new EnumValue({
						value: val
					})
				})
			]);
			// actionPending = false;
		}
	};

	let comboOpen = false;

	// We want to refocus the trigger button when the user selects
	// an item from the list so users can continue navigating the
	// rest of the form with the keyboard.
	function closeAndFocusTrigger(currentValue: any, triggerId: string) {
		action(currentValue);
		comboOpen = false;
		tick().then(() => {
			document.getElementById(triggerId)?.focus();
		});
	}
</script>

{#if service.typ === Service_ServiceType.ENUM}
<ServiceRoot deviceName={title} online={online} service={service}>
	<span slot="icon">
		<div class="p-2 rounded-full bg-secondary text-secondary-foreground">
			{#if true}
			<Rows3 />
			{/if}
		</div>
	</span>
	<span slot="details">
		{#if attrValue !== undefined}
			{#if attrValue.value === ""}
			<p>
				No state
			</p>
			{:else}
			<p>
				{attrValue.value}
			</p>
			{/if}
		{/if}
		{#if attrValue !== undefined && showOptions}
			<p class="text-muted-foreground">
				{"["}
				{#each attrValue.options as opt, index}
					{#if index > 0}{", "}{/if}
					{opt}
				{/each}
				{"]"}
			</p>
		{/if}
	</span>
	<span slot="dialog-desktop">
		{#if attrValue !== undefined}
			<Popover.Root bind:open={comboOpen} let:ids>
				<Popover.Trigger asChild let:builder>
					<Button
						builders={[builder]}
						variant="outline"
						role="combobox"
						aria-expanded={open}
						class="w-full justify-between"
					>
						{attrValue.value}
						<ChevronsUpDown class="ml-2 h-4 w-4 shrink-0 opacity-50" />
					</Button>
				</Popover.Trigger>
				<Popover.Content class=" p-0">
					<Command.Root>
						<Command.Input placeholder="Search options..." />
						<Command.Empty>No option found.</Command.Empty>
						<Command.Group>
						{#each attrValue.options as option}
							<Command.Item
								value={option}
								onSelect={(currentValue) => closeAndFocusTrigger(currentValue, ids.trigger)}
							>
							<Check
								class={cn(
								"mr-2 h-4 w-4",
								attrValue.value !== option && "text-transparent"
								)}
							/>
							{option}
							</Command.Item>
						{/each}
						</Command.Group>
					</Command.Root>
				</Popover.Content>
			</Popover.Root>
		{/if}
	</span>
	<span slot="dialog-mobile">
		{#if attrValue !== undefined}
			<Popover.Root bind:open={comboOpen} let:ids>
				<Popover.Trigger asChild let:builder>
					<Button
						builders={[builder]}
						variant="outline"
						role="combobox"
						aria-expanded={open}
						class="w-full justify-between"
					>
						{attrValue.value}
						<ChevronsUpDown class="ml-2 h-4 w-4 shrink-0 opacity-50" />
					</Button>
				</Popover.Trigger>
				<Popover.Content class="p-0">
					<Command.Root>
						<!-- <Command.Input placeholder="Search options..." /> -->
						<Command.Empty>No option found.</Command.Empty>
						<Command.Group>
						{#each attrValue.options as option}
							<Command.Item
								value={option}
								onSelect={(currentValue) => closeAndFocusTrigger(currentValue, ids.trigger)}
							>
							<Check
								class={cn(
								"mr-2 h-4 w-4",
								attrValue.value !== option && "text-transparent"
								)}
							/>
							{option}
							</Command.Item>
						{/each}
						</Command.Group>
					</Command.Root>
				</Popover.Content>
			</Popover.Root>
		{/if}
	</span>
</ServiceRoot>
{:else}
	<p>ERROR Service Type {Service_ServiceType[service.typ]} is not ENUM</p>
{/if}
