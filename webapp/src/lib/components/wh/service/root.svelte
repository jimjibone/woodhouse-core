<script lang="ts">
	import { cn } from "$lib/utils.js";
	import { mediaQuery } from "svelte-legos";
	import * as Dialog from "$lib/components/ui/dialog/index.js";
	import * as Drawer from "$lib/components/ui/drawer";

	export let title: string = "";
	export let alias: string = "";
	export let online: boolean;

	$: cardTitle = title ? title + (alias !== '' ? ': ' + alias : '') : alias;

	const isDesktop = mediaQuery("(min-width: 768px)");
	let drawerOpen: boolean = false;
	let openDrawer = () => {
		drawerOpen = true;
	};
	let closeDrawer = () => {
		drawerOpen = false;
	};
</script>

<button class={cn('rounded-lg border bg-card p-2 text-card-foreground shadow-sm text-left', !online && 'bg-muted')} on:click={openDrawer}>
	<div class="flex flex-row gap-2">
		<div class="shrink">
			<div class="grid h-full place-content-center">
				<slot name="icon">
					<p>No icon</p>
				</slot>
			</div>
		</div>
		<div class="grow">
			<div class="flex h-full flex-col justify-center gap-0">
				{#if cardTitle !== ''}
					<div class="rounded-lg p-0">
						<p class="font-semibold">{cardTitle}</p>
					</div>
				{/if}
				<div class="flex flex-row gap-2 rounded-lg p-0">
					<slot name="details">
						<p>No content</p>
					</slot>
				</div>
			</div>
		</div>
	</div>
</button>

{#if $isDesktop}
<Dialog.Root bind:open={drawerOpen}>
	<Dialog.Content class={!online && 'bg-muted'}>
		<Dialog.Header>
			<Dialog.Title>{cardTitle}</Dialog.Title>
			<!-- <Dialog.Description>This action cannot be undone.</Dialog.Description> -->
		</Dialog.Header>
		<slot name="dialog-desktop">
			<p>No content for <code>dialog-desktop</code></p>
		</slot>
	</Dialog.Content>
  </Dialog.Root>
{:else}
<Drawer.Root bind:open={drawerOpen}>
	<Drawer.Content class={cn("max-h-[96%]", !online && 'bg-muted')}>
		<div class="w-full mx-auto flex flex-col overflow-auto p-4 rounded-t-[10px] ">
			<Drawer.Header>
				<Drawer.Title>{cardTitle}</Drawer.Title>
				<!-- <Drawer.Description>This action cannot be undone.</Drawer.Description> -->
			</Drawer.Header>
			<slot name="dialog-mobile">
				<p>No content for <code>dialog-mobile</code></p>
			</slot>
		</div>
	</Drawer.Content>
</Drawer.Root>
{/if}
