<script lang="ts">
	import type { EnumAttribute } from '$lib/api/v1/clients/client_service_pb';
	import * as Command from "$lib/components/ui/command";
	import * as Popover from "$lib/components/ui/popover";
	import { Button } from "$lib/components/ui/button";
	import { tick } from "svelte";
	import { ChevronsUpDownIcon, CheckIcon } from '@lucide/svelte';
	import { cn } from "$lib/utils.js";
	import { toast } from "svelte-sonner";

	let {
		name,
		attr,
		onaction = undefined,
		class: className = ""
	}: {
		name: string,
		attr: EnumAttribute,
		onaction?: (value: string)=>void,
		class?: string
	} = $props();

	let open = $state(false);
	let triggerRef = $state<HTMLButtonElement>(null!);

	async function copyToClipboard(val: string) {
		try {
			await navigator.clipboard.writeText(val);
			toast.info(`Copied ${val} to clipboard`);
		} catch (err) {
			toast.error(`Failed to copy ${val} to clipboard: ` + err, {duration: 30000});
			console.error(`Failed to copy ${val} to clipboard: `, err);
		}
	}

	// We want to refocus the trigger button when the user selects
	// an item from the list so users can continue navigating the
	// rest of the form with the keyboard.
	function closeAndFocusTrigger() {
		open = false;
		tick().then(() => {
			triggerRef.focus();
		});
	}
</script>

<div class={className}>{name}</div>
<div class="col-span-2 flex flex-row gap-0">
	<Popover.Root bind:open>
		<Popover.Trigger bind:ref={triggerRef} class="">
			{#snippet child({ props })}
			<Button
				{...props}
				variant="outline"
				class="w-full justify-between cursor-pointer"
				role="combobox"
				aria-expanded={open}
			>
				{attr.value || "Select an option..."}
				<ChevronsUpDownIcon class="opacity-50" />
			</Button>
			{/snippet}
		</Popover.Trigger>
		<Popover.Content class="w-[var(--bits-popover-anchor-width)] min-w-[var(--bits-popover-anchor-width)] p-0">
			<Command.Root>
				<!-- <Command.Input placeholder="Search options..." /> -->
				<Command.List>
					<Command.Empty>No framework found.</Command.Empty>
					<Command.Group value="options">
						<Command.Item disabled>
							<CheckIcon class="text-transparent"/>
							Read Only - Select to Copy
						</Command.Item>
						{#each attr.options as option (option)}
							<Command.Item
								value={option}
								onSelect={() => {
									if (onaction) {
										onaction(option);
									} else {
										copyToClipboard(option);
									}
									closeAndFocusTrigger();
								}}
							>
								<CheckIcon
									class={cn(attr.value !== option && "text-transparent")}
								/>
								{option}
							</Command.Item>
						{/each}
					</Command.Group>
				</Command.List>
			</Command.Root>
		</Popover.Content>
	</Popover.Root>
</div>
