<script lang="ts">
	import { type Snippet } from 'svelte';
	import * as Dialog from '$lib/components/ui/dialog';
	import * as Drawer from '$lib/components/ui/drawer';
	import { IsMobile } from '$lib/hooks/is-mobile.svelte.js';

	let {
		title = undefined,
		open = $bindable(false),
		children = undefined
	}: {
		title?: string | undefined;
		open: boolean;
		children?: Snippet<[]>;
	} = $props();

	const isMobile = new IsMobile();
</script>

{#if isMobile.current}
	<Drawer.Root bind:open>
		<Drawer.Content class="min-h-[50%]">
			<div class="w-full mx-auto flex flex-col overflow-auto p-4 rounded-t-[10px] pt-0">
				{#if title}
					<Drawer.Header>
						<Drawer.Title class="text-lg">
							{title}
						</Drawer.Title>
					</Drawer.Header>
				{/if}

				<div class="px-4 pb-4 flex flex-col">
					{#if children}
						{@render children()}
					{:else}
						<p>No content</p>
					{/if}
				</div>
			</div>
		</Drawer.Content>
	</Drawer.Root>
{:else}
	<Dialog.Root bind:open>
		<Dialog.Content class="max-h-[90%] overflow-y-auto" showCloseButton={false}>
			<div class="grid gap-1">
				{#if title}
					<Dialog.Header>
						<Dialog.Title class="text-lg">
							{title}
						</Dialog.Title>
					</Dialog.Header>
					<div class="pt-4"></div>
				{/if}

				{#if children}
					{@render children()}
				{:else}
					<p>No content</p>
				{/if}
			</div>
		</Dialog.Content>
	</Dialog.Root>
{/if}
