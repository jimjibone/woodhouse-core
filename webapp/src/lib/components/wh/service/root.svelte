<script lang="ts">
	import { cn } from "$lib/utils.js";
	import { mediaQuery } from "svelte-legos";
	import { Service } from '$lib/api/v1/clients/client_service_pb';
	import * as Dialog from "$lib/components/ui/dialog/index.js";
	import * as Drawer from "$lib/components/ui/drawer";
	import { Button } from "$lib/components/ui/button";
	import { Heart, HeartOff, Eye, EyeOff } from 'lucide-svelte';
	import LittleButton from './little-button.svelte'

	export let deviceName: string = "";
	export let online: boolean;
	export let service: Service;
	export let expandable: boolean = false;
	export let onSetFavorite: ((serviceID: string, fave: boolean) => Promise<void>) | undefined = undefined;

	$: cardTitle = deviceName ? deviceName + (service.alias !== '' ? ': ' + service.alias : '') : service.alias;

	const isDesktop = mediaQuery("(min-width: 768px)");

	let drawerOpen: boolean = false;
	let openDrawer = () => {
		drawerOpen = true;
	};

	let expanded: boolean = false;

	let toggleFavorite = async() => {
		if (onSetFavorite) {
			await onSetFavorite(service.id, !service.favorite);
		}
	}
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
					<div class="rounded-lg p-0 flex flex-row items-center">
						<p class="font-semibold">{cardTitle}</p>
						{#if service.favorite}<Heart class="h-4 ml-2" />{/if}
					</div>
				{/if}
				<slot name="details" expanded={expanded} class="flex flex-row gap-2 rounded-lg p-0">
					<p>No content</p>
				</slot>
			</div>
		</div>
		{#if expandable}
			<div class="shrink">
				<div class="grid h-full place-content-center">
					<LittleButton on:click={() => expanded = !expanded}>
					{#if expanded}
						<EyeOff class="size-4" />
					{:else}
						<Eye class="size-4" />
					{/if}
					</LittleButton>
				</div>
			</div>
		{/if}
	</div>
	{#if $isDesktop}
	<Dialog.Root bind:open={drawerOpen}>
		<Dialog.Content class={!online && 'bg-muted'}>
			<Dialog.Header>
				<Dialog.Title>
					{cardTitle}
					<LittleButton on:click={toggleFavorite}>
						{#if service.favorite}
						<Heart class="h-4 w-4" />
						{:else}
						<HeartOff class="h-4 w-4" />
						{/if}
					</LittleButton>
				</Dialog.Title>
				<!-- <Dialog.Description>This action cannot be undone.</Dialog.Description> -->
			</Dialog.Header>
			<slot name="dialog-desktop">
				<!-- <p>No content for <code>dialog-desktop</code></p> -->
			</slot>
		</Dialog.Content>
	  </Dialog.Root>
	{:else}
	<Drawer.Root bind:open={drawerOpen}>
		<Drawer.Content class={cn("max-h-[96%]", !online && 'bg-muted')}>
			<div class="w-full mx-auto flex flex-col overflow-auto p-4 rounded-t-[10px] ">
				<Drawer.Header>
					<Drawer.Title>
						{cardTitle}
						<LittleButton on:click={toggleFavorite}>
							{#if service.favorite}
							<Heart class="h-4 w-4" />
							{:else}
							<HeartOff class="h-4 w-4" />
							{/if}
						</LittleButton>
					</Drawer.Title>
				</Drawer.Header>
				<slot name="dialog-mobile">
					<!-- <p>No content for <code>dialog-mobile</code></p> -->
				</slot>
			</div>
		</Drawer.Content>
	</Drawer.Root>
	{/if}
</button>
