<script lang="ts">
	import CheckIcon from '@lucide/svelte/icons/check';
	import ChevronsUpDownIcon from '@lucide/svelte/icons/chevrons-up-down';
	import { tick } from 'svelte';
	import * as Command from '$lib/components/ui/command/index.js';
	import * as Popover from '$lib/components/ui/popover/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { cn } from '$lib/utils.js';

	export type ComboOption = {
		value: any;
		label: string;
	};

	let {
		options,
		value = $bindable(),
		withSearch = false
	}: {
		options: ComboOption[];
		value: any;
		withSearch?: boolean;
	} = $props();

	let open = $state(false);
	let triggerRef = $state<HTMLButtonElement>(null!);

	const selectedLabel = $derived(options.find((f) => f.value === value)?.label);

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

<Popover.Root bind:open>
	<Popover.Trigger bind:ref={triggerRef}>
		{#snippet child({ props })}
			<Button
				{...props}
				variant="outline"
				class="w-[200px] justify-between cursor-pointer"
				role="combobox"
				aria-expanded={open}
			>
				{selectedLabel || 'Select...'}
				<ChevronsUpDownIcon class="opacity-50" />
			</Button>
		{/snippet}
	</Popover.Trigger>
	<Popover.Content class="w-[200px] p-0">
		<Command.Root>
			{#if withSearch}
				<Command.Input placeholder="Search..." />
			{/if}
			<Command.List>
				<Command.Empty>No options found.</Command.Empty>
				<Command.Group value="options">
					{#each options as option (option.value)}
						<Command.Item
							value={option.label}
							onSelect={() => {
								value = option.value;
								closeAndFocusTrigger();
							}}
						>
							<CheckIcon class={cn(value !== option.value && 'text-transparent')} />
							{option.label}
						</Command.Item>
					{/each}
				</Command.Group>
			</Command.List>
		</Command.Root>
	</Popover.Content>
</Popover.Root>
